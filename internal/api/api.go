package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/mikoim/go-loadavg"
	"github.com/serverledge-faas/serverledge/internal/client"
	"github.com/serverledge-faas/serverledge/internal/function"
	"github.com/serverledge-faas/serverledge/internal/node"
	"github.com/serverledge-faas/serverledge/internal/registration"
	"github.com/serverledge-faas/serverledge/internal/scheduling"
	"github.com/serverledge-faas/serverledge/internal/telemetry"
	"github.com/serverledge-faas/serverledge/internal/variants"
	"github.com/serverledge-faas/serverledge/internal/workflow"
	"github.com/serverledge-faas/serverledge/utils"
	"go.opentelemetry.io/otel/attribute"
)

var requestsPool = sync.Pool{
	New: func() any {
		return new(function.Request)
	},
}

var workflowInvocationRequestPool = sync.Pool{
	New: func() any {
		return new(workflow.Request)
	},
}

// GetFunctions handles a request to list the function available in the system.
func GetFunctions(c echo.Context) error {
	list, err := function.GetAll()
	if err != nil {
		return c.String(http.StatusServiceUnavailable, "")
	}
	return c.JSON(http.StatusOK, list)
}

// InvokeFunction handles a function invocation request.
func InvokeFunction(c echo.Context) error {
	funcName := c.Param("fun")

	// =====================================================
	// 1. Lookup funzione
	// =====================================================
	fun, ok := function.GetFunction(funcName)
	if !ok {
		log.Printf("Dropping request for unknown function '%s'\n", funcName)
		return c.String(http.StatusNotFound, "Function unknown")
	}

	// =====================================================
	// 2. Parse request body
	// =====================================================
	var invocationRequest client.InvocationRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&invocationRequest); err != nil && err != io.EOF {
		log.Printf("Could not parse request: %v\n", err)
		return c.String(http.StatusBadRequest, "Invalid invocation request")
	}

	// =====================================================
	// 3. Build internal request
	// =====================================================
	r := requestsPool.Get().(*function.Request)
	defer requestsPool.Put(r)

	r.Fun = fun
	r.Params = invocationRequest.Params
	r.Arrival = time.Now()
	r.MaxRespT = invocationRequest.QoSMaxRespT
	r.CanDoOffloading = invocationRequest.CanDoOffloading
	r.Async = invocationRequest.Async
	r.ReturnOutput = invocationRequest.ReturnOutput

	// Energy-aware fields
	r.AllowApprox = invocationRequest.AllowApprox
	r.MaxEnergyJoule = invocationRequest.MaxEnergyJoule

	reqId := fmt.Sprintf(
		"%s-%s-%d",
		funcName,
		node.LocalNode.String()[len(node.LocalNode.String())-5:],
		r.Arrival.Nanosecond(),
	)

	r.Ctx = context.WithValue(context.Background(), "ReqId", reqId)

	// =====================================================
	// 4. Tracing
	// =====================================================
	if telemetry.DefaultTracer != nil {
		ctx, span := telemetry.DefaultTracer.Start(r.Ctx, "invocation")
		r.Ctx = ctx
		span.SetAttributes(attribute.String("function", r.Fun.Name))
		defer span.End()
	}

	// =====================================================
	// 5. Async path
	// =====================================================
	if r.Async {
		go scheduling.SubmitAsyncRequest(r)
		return c.JSON(http.StatusOK, function.AsyncResponse{ReqId: r.Id()})
	}

	// =====================================================
	// 6. Sync execution
	// =====================================================
	executionReport, err := scheduling.SubmitRequest(r)
	if err != nil {

		// Caso nodo saturo
		if errors.Is(err, node.OutOfResourcesErr) {
			return c.String(http.StatusTooManyRequests, "Node out of resources")
		}

		// Caso errore di policy energetica
		if r.AllowApprox && r.MaxEnergyJoule != nil {

			log.Printf("Invocation rejected by energy policy: %v\n", err)

			// executionReport può essere nil
			var schedReport *function.VariantSchedulingReport
			if executionReport != nil {
				schedReport = executionReport.VariantSchedulingReport
			}

			return c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
				"success":            false,
				"error":              err.Error(),
				"variant_scheduling": schedReport,
			})
		}

		// Errore interno generico
		log.Printf("Invocation failed: %v\n", err)
		return c.String(http.StatusInternalServerError, "Internal server error")
	}

	// =====================================================
	// 7. Success
	// =====================================================
	return c.JSON(http.StatusOK, function.Response{
		Success:         true,
		ExecutionReport: *executionReport,
	})
}

// PollAsyncResult checks for the result of an asynchronous invocation.
func PollAsyncResult(c echo.Context) error {
	reqId := c.Param("reqId")
	if len(reqId) < 0 {
		return c.String(http.StatusNotFound, "")
	}

	etcdClient, err := utils.GetEtcdClient()
	if err != nil {
		log.Println("Could not connect to Etcd")
		return c.String(http.StatusInternalServerError, "Failed to connect to the Global Registry")
	}

	ctx := context.Background()

	key := fmt.Sprintf("async/%s", reqId)
	res, err := etcdClient.Get(ctx, key)
	if err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, "Could not retrieve results")
	}

	if len(res.Kvs) == 1 {
		payload := res.Kvs[0].Value
		return c.JSONBlob(http.StatusOK, payload)
	} else {
		return c.String(http.StatusNotFound, "")
	}
}

// CreateOrUpdateFunction handles a function creation/update request.
func CreateOrUpdateFunction(c echo.Context) error {

	log.Println("Parse Request")

	// ------------------------------------------------------------------
	// 1. Parse request (logical function + create-time flags)
	// ------------------------------------------------------------------
	var fn function.Function
	if err := json.NewDecoder(c.Request().Body).Decode(&fn); err != nil && err != io.EOF {
		log.Printf("Could not parse request: %v\n", err)
		return c.String(http.StatusBadRequest, "Invalid request body")
	}

	ctx := c.Request().Context()

	// ------------------------------------------------------------------
	// 2. Create vs Update
	// ------------------------------------------------------------------
	if c.Path() != "/update" {
		if _, ok := function.GetFunction(fn.Name); ok {
			return c.String(http.StatusConflict, "Function already exists")
		}
		log.Printf("Creating function %s\n", fn.Name)
	} else {
		log.Printf("Creating/updating function %s\n", fn.Name)
	}

	// ------------------------------------------------------------------
	// 3. Sanity checks (logical name only)
	// ------------------------------------------------------------------
	if fn.Name == "" {
		return c.String(http.StatusUnprocessableEntity, "Function name is required")
	}

	if fn.MemoryMB < 1 {
		return c.String(http.StatusUnprocessableEntity, "Invalid memory limit")
	}

	if fn.MaxConcurrency <= 0 {
		fn.MaxConcurrency = 1
	}

	// ------------------------------------------------------------------
	// 4. Load variants (MANDATORY: base is included in JSON)
	// ------------------------------------------------------------------

	if fn.AllowApprox {

		log.Println("AllowApprox enabled: loading variants")

		variantsDir := os.Getenv("SERVERLEDGE_VARIANTS_DIR")
		if variantsDir == "" {
			variantsDir = "variants"
		}

		variantFactory := &variants.Factory{
			FileSource: &variants.FileSource{
				BaseDir: variantsDir,
			},
			GeneratorSource: &variants.GeneratorSource{}, // future
		}

		source, err := variantFactory.GetSource(&fn)
		if err != nil {
			log.Printf("No variant source available: %v\n", err)
			return c.String(http.StatusInternalServerError, err.Error())
		}

		loadedVariants, err := source.Load(ctx, &fn)
		if err != nil {
			log.Printf("Failed loading variants: %v\n", err)
			return c.String(http.StatusNotFound, err.Error())
		}

		if len(loadedVariants) == 0 {
			return c.String(http.StatusUnprocessableEntity, "No variants provided")
		}

		if err := variants.ValidateVariantsOnly(loadedVariants); err != nil {
			log.Printf("Variant validation failed: %v\n", err)
			return c.String(http.StatusUnprocessableEntity, err.Error())
		}

		if err := variants.CreateInternalVariants(&fn, loadedVariants); err != nil {
			log.Printf("Failed creating variant functions: %v\n", err)
			return c.String(http.StatusInternalServerError, err.Error())
		}

	} else {
		// ------------------------------------------------------------------
		// LEGACY PATH (single function, NO variants)
		// ------------------------------------------------------------------
		log.Println("AllowApprox disabled: creating single function")

		fn.LogicalName = fn.Name
		fn.VariantID = "base"

		if err := fn.SaveToEtcd(); err != nil {
			log.Printf("Failed saving function: %v\n", err)
			return c.String(http.StatusServiceUnavailable, "Failed to save function")
		}
	}

	// ------------------------------------------------------------------
	// 6. Response
	// ------------------------------------------------------------------
	return c.JSON(http.StatusOK, struct {
		Created string `json:"created"`
	}{
		Created: fn.Name,
	})
}

// DeleteFunction handles a function deletion request.
func DeleteFunction(c echo.Context) error {
	var f function.Function
	err := json.NewDecoder(c.Request().Body).Decode(&f)
	if err != nil && err != io.EOF {
		log.Printf("Could not parse request: %v\n", err)
		return err
	}

	_, ok := function.GetFunction(f.Name) // TODO: we would need a system-wide lock here...
	if !ok {
		log.Printf("Dropping request for non existing function '%s'\n", f.Name)
		return c.String(http.StatusNotFound, "Unknown function")
	}

	log.Printf("New request: deleting %s\n", f.Name)
	err = function.DeleteLogicalFunction(f.Name)
	if err != nil {
		log.Printf("Failed deletion: %v\n", err)

		if errors.Is(err, errors.New("function not found")) {
			return c.String(http.StatusNotFound, "Unknown function")
		}

		return c.String(http.StatusServiceUnavailable, "")
	}

	// Delete local warm containers
	node.ShutdownWarmContainersFor(&f)

	response := struct{ Deleted string }{f.Name}
	return c.JSON(http.StatusOK, response)
}

// GetServerStatus simple api to check the current server status
func GetServerStatus(c echo.Context) error {
	node.LocalResources.RLock()
	defer node.LocalResources.RUnlock()

	loadAvg, err := loadavg.Parse()
	loadAvgValues := []float64{-1.0, -1.0, -1.0}
	if err == nil {
		loadAvgValues = []float64{loadAvg.LoadAverage1, loadAvg.LoadAverage5, loadAvg.LoadAverage10}
	}

	// TODO: use a different type
	response := registration.StatusInformation{
		AvailableWarmContainers: node.WarmStatus(),
		TotalMemory:             node.LocalResources.TotalMemory(),
		UsedMemory:              node.LocalResources.UsedMemory(),
		TotalCPU:                node.LocalResources.TotalCPUs(),
		UsedCPU:                 node.LocalResources.UsedCPUs(),
		Coordinates:             *registration.VivaldiClient.GetCoordinate(),
		LoadAvg:                 loadAvgValues,
	}

	return c.JSON(http.StatusOK, response)
}

// PrewarmFunction handles a prewarming request.
func PrewarmFunction(c echo.Context) error {
	var req client.PrewarmingRequest
	err := json.NewDecoder(c.Request().Body).Decode(&req)
	if err != nil && err != io.EOF {
		log.Printf("Could not parse request: %v\n", err)
		return err
	}

	fun, ok := function.GetFunction(req.Function)
	if !ok {
		log.Printf("Dropping request for unknown fun '%s'\n", req.Function)
		return c.String(http.StatusNotFound, "Function unknown")
	}

	count, err := node.PrewarmInstances(fun, req.Instances, req.ForceImagePull)

	if err != nil && !errors.Is(err, node.OutOfResourcesErr) {
		log.Printf("Failed prewarming: %v\n", err)
		return c.JSON(http.StatusServiceUnavailable, "")
	}
	response := struct{ Prewarmed int64 }{count}
	return c.JSON(http.StatusOK, response)
}

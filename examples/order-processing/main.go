// Order Processing Example
//
// Demonstrates using the GCW emulator with the official Google Cloud Go client
// libraries via gRPC. This service:
//
//  1. Deploys an order-processing workflow to the emulator on startup.
//  2. Exposes step endpoints that the workflow calls via http.post.
//  3. Provides REST API endpoints to start workflows and check progress/results.
//
// Usage:
//
//	# Terminal 1: start the emulator
//	gcw-emulator --grpc-port=8788 --port=8787
//
//	# Terminal 2: start this service
//	WORKFLOWS_EMULATOR_HOST=localhost:8788 go run .
//
//	# Terminal 3: create an order
//	curl -s -X POST http://localhost:4000/api/orders -H 'Content-Type: application/json' -d '{
//	  "id": "ord-001",
//	  "items": [{"name":"Widget","quantity":2,"price":9.99},{"name":"Gadget","quantity":1,"price":24.99}],
//	  "customer": {"email":"alice@example.com"}
//	}'
//
//	# Check progress
//	curl -s http://localhost:4000/api/orders/ord-001/status | jq
//
//	# Get final result
//	curl -s http://localhost:4000/api/orders/ord-001/result | jq
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	workflows "cloud.google.com/go/workflows/apiv1"
	workflowspb "cloud.google.com/go/workflows/apiv1/workflowspb"
	executions "cloud.google.com/go/workflows/executions/apiv1"
	executionspb "cloud.google.com/go/workflows/executions/apiv1/executionspb"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

//go:embed workflow.yaml
var workflowSource string

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// orderTracker stores step-by-step progress for each order. The workflow calls
// back to our step endpoints, which record progress here. This lets the API
// return which steps have completed even while the workflow is still running.
type orderTracker struct {
	mu     sync.Mutex
	orders map[string]*orderProgress
}

type orderProgress struct {
	ExecutionName string      `json:"executionName"`
	Steps         []stepEvent `json:"steps"`
}

type stepEvent struct {
	Name      string    `json:"name"`
	Detail    string    `json:"detail,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

var tracker = &orderTracker{orders: make(map[string]*orderProgress)}

func (t *orderTracker) init(orderID, execName string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.orders[orderID] = &orderProgress{ExecutionName: execName}
}

func (t *orderTracker) record(orderID, stepName, detail string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if p, ok := t.orders[orderID]; ok {
		p.Steps = append(p.Steps, stepEvent{
			Name:      stepName,
			Detail:    detail,
			Timestamp: time.Now(),
		})
	}
}

func (t *orderTracker) get(orderID string) (*orderProgress, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	p, ok := t.orders[orderID]
	return p, ok
}

func main() {
	ctx := context.Background()

	// Configuration via environment variables.
	emulatorHost := envOr("WORKFLOWS_EMULATOR_HOST", "localhost:8788")
	project := envOr("PROJECT", "my-project")
	location := envOr("LOCATION", "us-central1")
	port := envOr("PORT", "4000")
	serviceURL := envOr("SERVICE_URL", "http://localhost:"+port)

	parent := fmt.Sprintf("projects/%s/locations/%s", project, location)
	workflowName := parent + "/workflows/order-processing"

	// Connect to the emulator's gRPC endpoint. The three options below are the
	// key difference from connecting to production GCP: we point at the local
	// emulator, skip authentication, and use insecure (plaintext) transport.
	grpcOpts := []option.ClientOption{
		option.WithEndpoint(emulatorHost),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	}

	// --- Deploy the workflow ---

	wfClient, err := workflows.NewClient(ctx, grpcOpts...)
	if err != nil {
		log.Fatalf("workflows client: %v", err)
	}
	defer wfClient.Close()

	log.Printf("Deploying workflow to emulator at %s ...", emulatorHost)
	op, err := wfClient.CreateWorkflow(ctx, &workflowspb.CreateWorkflowRequest{
		Parent:     parent,
		WorkflowId: "order-processing",
		Workflow: &workflowspb.Workflow{
			SourceCode: &workflowspb.Workflow_SourceContents{
				SourceContents: workflowSource,
			},
		},
	})
	if err != nil {
		log.Fatalf("deploy workflow: %v", err)
	}
	if _, err := op.Wait(ctx); err != nil {
		log.Fatalf("deploy workflow wait: %v", err)
	}
	log.Println("Workflow deployed successfully")

	// Create an executions client for starting and polling workflow runs.
	exClient, err := executions.NewClient(ctx, grpcOpts...)
	if err != nil {
		log.Fatalf("executions client: %v", err)
	}
	defer exClient.Close()

	// --- Fiber app ---

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(logger.New())

	// Step endpoints -- these are called by the workflow via http.post.
	registerStepEndpoints(app)

	// API endpoints -- these are called by the user/frontend.
	registerAPIEndpoints(app, exClient, workflowName, serviceURL)

	log.Printf("Order service listening on :%s  (emulator gRPC: %s)", port, emulatorHost)
	log.Fatal(app.Listen(":" + port))
}

// ---------------------------------------------------------------------------
// Step endpoints (called by the workflow)
// ---------------------------------------------------------------------------

func registerStepEndpoints(app *fiber.App) {
	// POST /steps/validate -- validate order items and compute total.
	app.Post("/steps/validate", func(c *fiber.Ctx) error {
		var req struct {
			OrderID string `json:"orderId"`
			Items   []struct {
				Name     string  `json:"name"`
				Quantity int     `json:"quantity"`
				Price    float64 `json:"price"`
			} `json:"items"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"valid": false, "error": "invalid request body"})
		}
		if len(req.Items) == 0 {
			return c.JSON(fiber.Map{"valid": false, "error": "order has no items"})
		}

		var total float64
		for _, item := range req.Items {
			total += float64(item.Quantity) * item.Price
		}

		tracker.record(req.OrderID, "validate_order", fmt.Sprintf("%.2f", total))
		return c.JSON(fiber.Map{"valid": true, "total": total})
	})

	// POST /steps/charge -- simulate payment processing.
	app.Post("/steps/charge", func(c *fiber.Ctx) error {
		var req struct {
			OrderID string  `json:"orderId"`
			Amount  float64 `json:"amount"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
		}

		txnID := fmt.Sprintf("txn-%06d", rand.Intn(1_000_000))
		tracker.record(req.OrderID, "process_payment", fmt.Sprintf("charged %.2f -> %s", req.Amount, txnID))
		return c.JSON(fiber.Map{"transactionId": txnID, "charged": req.Amount})
	})

	// POST /steps/confirm -- simulate sending a confirmation email.
	app.Post("/steps/confirm", func(c *fiber.Ctx) error {
		var req struct {
			OrderID       string  `json:"orderId"`
			Email         string  `json:"email"`
			TransactionID string  `json:"transactionId"`
			Total         float64 `json:"total"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
		}

		tracker.record(req.OrderID, "send_confirmation", fmt.Sprintf("email=%s", req.Email))
		log.Printf("[confirm] Order %s: confirmation sent to %s (txn %s, $%.2f)",
			req.OrderID, req.Email, req.TransactionID, req.Total)
		return c.JSON(fiber.Map{"sent": true})
	})
}

// ---------------------------------------------------------------------------
// API endpoints (called by user/frontend)
// ---------------------------------------------------------------------------

func registerAPIEndpoints(app *fiber.App, exClient *executions.Client, workflowName, serviceURL string) {
	// POST /api/orders -- start a new order workflow.
	app.Post("/api/orders", func(c *fiber.Ctx) error {
		var order struct {
			ID       string      `json:"id"`
			Items    any `json:"items"`
			Customer struct {
				Email string `json:"email"`
			} `json:"customer"`
		}
		if err := c.BodyParser(&order); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid order payload"})
		}
		if order.ID == "" {
			order.ID = fmt.Sprintf("ord-%06d", rand.Intn(1_000_000))
		}

		args, _ := json.Marshal(map[string]any{
			"serviceUrl": serviceURL,
			"order": map[string]any{
				"id":       order.ID,
				"items":    order.Items,
				"customer": map[string]any{"email": order.Customer.Email},
			},
		})

		exec, err := exClient.CreateExecution(c.Context(), &executionspb.CreateExecutionRequest{
			Parent: workflowName,
			Execution: &executionspb.Execution{
				Argument: string(args),
			},
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		tracker.init(order.ID, exec.GetName())
		return c.Status(201).JSON(fiber.Map{
			"orderId":       order.ID,
			"executionName": exec.GetName(),
			"state":         "ACTIVE",
		})
	})

	// GET /api/orders/:id/status -- step-by-step progress + workflow state.
	app.Get("/api/orders/:id/status", func(c *fiber.Ctx) error {
		orderID := c.Params("id")
		progress, ok := tracker.get(orderID)
		if !ok {
			return c.Status(404).JSON(fiber.Map{"error": "order not found"})
		}

		exec, err := exClient.GetExecution(c.Context(), &executionspb.GetExecutionRequest{
			Name: progress.ExecutionName,
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{
			"orderId":        orderID,
			"workflowState":  exec.GetState().String(),
			"stepsCompleted": progress.Steps,
		})
	})

	// GET /api/orders/:id/result -- final workflow result (or error).
	app.Get("/api/orders/:id/result", func(c *fiber.Ctx) error {
		orderID := c.Params("id")
		progress, ok := tracker.get(orderID)
		if !ok {
			return c.Status(404).JSON(fiber.Map{"error": "order not found"})
		}

		exec, err := exClient.GetExecution(c.Context(), &executionspb.GetExecutionRequest{
			Name: progress.ExecutionName,
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		state := exec.GetState().String()
		switch exec.GetState() {
		case executionspb.Execution_SUCCEEDED:
			var result any
			json.Unmarshal([]byte(exec.GetResult()), &result)
			return c.JSON(fiber.Map{
				"orderId": orderID,
				"state":   state,
				"result":  result,
			})
		case executionspb.Execution_FAILED:
			return c.JSON(fiber.Map{
				"orderId": orderID,
				"state":   state,
				"error":   exec.GetError().GetPayload(),
			})
		default:
			return c.JSON(fiber.Map{
				"orderId": orderID,
				"state":   state,
				"message": "workflow is still running",
			})
		}
	})
}

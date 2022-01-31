package main

import (
	"fmt"
	"net/http"
	"context"
	"log"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

func newExporter(ctx context.Context) (*otlptrace.Exporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint("api.honeycomb.io:443"),
		otlptracegrpc.WithHeaders(map[string]string{
			"x-honeycomb-team":    os.Getenv("HONEYCOMB_API_KEY"),
			"x-honeycomb-dataset": os.Getenv("HONEYCOMB_DATASET"),
		}),
		otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")),
	}

	client := otlptracegrpc.NewClient(opts...)
	return otlptrace.New(ctx, client)
}

func newTraceProvider(exp *otlptrace.Exporter) *sdktrace.TracerProvider {
    // The service.name attribute is required.
	resource :=
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("ExampleService"),
		)

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource),
	)
}

func index(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "apa kabar!")
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, World")
}

func main() {
    

	ctx := context.Background()

	// Configure a new exporter using environment variables for sending data to Honeycomb over gRPC.
	exp, err := newExporter(ctx)
	if err != nil {
		log.Fatalf("failed to initialize exporter: %v", err)
	}

	// Create a new tracer provider with a batch span processor and the otlp exporter.
	tp := newTraceProvider(exp)

	// Handle this error in a sensible manner where possible
	defer func() { _ = tp.Shutdown(ctx) }()

	// Set the Tracer Provider and the W3C Trace Context propagator as globals
	otel.SetTracerProvider(tp)

    // Register the trace context and baggage propagators so data is propagated across services/processes.
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "halo!")
    })

    http.HandleFunc("/index", index)

	handler := http.HandlerFunc(httpHandler)
	wrappedHandler := otelhttp.NewHandler(handler, "hello")
	http.Handle("/hello", wrappedHandler)

    fmt.Println("starting web server at http://localhost:8080/")
    http.ListenAndServe(":8080", nil)
}
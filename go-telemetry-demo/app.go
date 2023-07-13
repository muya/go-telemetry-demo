package main

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"io"
	"log"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

const name = "fib"

// App is a Fibonacci computation application.
type App struct {
	r io.Reader
	l *log.Logger
}

// NewApp returns a new App.
func NewApp(r io.Reader, l *log.Logger) *App {
	return &App{r: r, l: l}
}

// Run starts polling users for Fibonacci number requests and writes results.
func (a *App) Run(ctx context.Context) error {
	for {
		// Each execution of the run loop, we should get a new "root" span and context
		newCtx, span := otel.Tracer(name).Start(ctx, "Run")

		n, pollErr := a.Poll(newCtx)
		if pollErr != nil {
			span.End()
			return pollErr
		}

		a.Write(ctx, n)
		span.End()
	}
}

// Poll asks a user for input and returns the request.
func (a *App) Poll(ctx context.Context) (uint, error) {
	_, span := otel.Tracer(name).Start(ctx, "Poll")
	defer span.End()

	a.l.Print("What Fibonacci number would you like to know: ")

	var n uint
	_, scanErr := fmt.Fscanf(a.r, "%d\n", &n)

	if scanErr != nil {
		span.RecordError(scanErr)
		span.SetStatus(codes.Error, scanErr.Error())
		return 0, scanErr
	}

	// Store n as a string to not overflow an int64
	nStr := strconv.FormatUint(uint64(n), 10)
	span.SetAttributes(attribute.String("request.n", nStr))

	return n, nil
}

// Write writes the n-th Fibonacci number back to the user.
func (a *App) Write(ctx context.Context, n uint) {
	var span trace.Span
	ctx, span = otel.Tracer(name).Start(ctx, "Write")
	defer span.End()

	f, calculateErr := func(ctx context.Context) (uint64, error) {
		_, currentSpan := otel.Tracer(name).Start(ctx, "Fibonacci")
		defer currentSpan.End()

		f, fibErr := Fibonacci(n)
		if fibErr != nil {
			currentSpan.RecordError(fibErr)
			currentSpan.SetStatus(codes.Error, fibErr.Error())
		}

		return f, fibErr
	}(ctx)

	if calculateErr != nil {
		a.l.Printf("Fibonacci(%d): %v\n", n, calculateErr)
	} else {
		a.l.Printf("Fibonacci(%d) = %d\n", n, f)
	}
}

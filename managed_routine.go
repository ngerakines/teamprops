package teamprops

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ManagedRoutine interface {
	Run() error
	Shutdown(context.Context) error
}

func RunManagedRoutine(server ManagedRoutine, stop chan struct{}, wg *sync.WaitGroup) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		wg.Add(1)
		<-stop
		fmt.Println("Gracefully stopping.")
		if err := server.Shutdown(ctx); err != nil {
			fmt.Println("Error gracefully stopping.")
		} else {
			fmt.Println("Gracefully stopped.")
		}
		wg.Done()
	}()

	return server.Run()
}


package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/metrics"
	"time"
)

const blockSize = 1 << 20 // 1 MiB

func main() {
	liveMiB := flag.Int("live", 256, "Объем сохраняемой в памяти данных MiB")
	churnMiB := flag.Int("churn", 32, "Объем временных выделений памяти за один цикл MiB")
	rounds := flag.Int("rounds", 8, "Количество циклов выделения памяти")
	limitMiB := flag.Int("limit", 384, "Лимит памяти MiB; 0 лимит отключен")

	flag.Parse()

	if *liveMiB < 0 || *churnMiB < 0 || *rounds < 1 || *limitMiB < 0 {
		fmt.Fprintln(os.Stderr, "недопустимые аргументы")
		os.Exit(1)
	}

	if *limitMiB > 0 {
		previous := debug.SetMemoryLimit(int64(*limitMiB) << 20)
		fmt.Printf("Ограничение памяти: %d MiB (предыдущее: %d байт)\n",
			*limitMiB, previous)
	} else {
		debug.SetMemoryLimit(-1)
		fmt.Println("Лимит памяти: отключен")
	}

	fmt.Printf("GOGC: %s\n", envOrDefault("GOGC", "100 (default)"))
	fmt.Printf("Оперативная память: %d MiB\n", *liveMiB)
	fmt.Printf("Временные распределения на раунд: %d MiB\n", *churnMiB)
	fmt.Printf("rounds: %d\n\n", *rounds)

	live := make([]byte, *liveMiB*blockSize)
	touch(live)

	before := readStats()

	start := time.Now()

	for round := 1; round <= *rounds; round++ {
		temporary := make([][]byte, *churnMiB)

		for i := range temporary {
			temporary[i] = make([]byte, blockSize)
			touch(temporary[i])
		}

		runtime.KeepAlive(temporary)
		temporary = nil

		fmt.Printf("%d/%d выполнено\n", round, *rounds)
	}

	elapsed := time.Since(start)
	after := readStats()

	runtime.KeepAlive(live)

	fmt.Println("\n--- Результаты ---")
	fmt.Printf("прошло:          %s\n", elapsed)
	fmt.Printf("GC cycles:        %d\n", after.NumGC-before.NumGC)
	fmt.Printf("GC pause: %.3f ms\n", float64(after.PauseTotalNs-before.PauseTotalNs)/1e6)
	fmt.Printf("HeapAlloc до: %d MiB\n", before.HeapAlloc>>20)
	fmt.Printf("HeapAlloc после:  %d MiB\n", after.HeapAlloc>>20)
	fmt.Printf("HeapSys после:    %d MiB\n", after.HeapSys>>20)
	fmt.Printf("TotalAlloc:       %d MiB\n",
		(after.TotalAlloc-before.TotalAlloc)>>20)
	fmt.Printf("GC CPU fraction:  %.4f\n", after.GCCPUFraction)
}

func touch(data []byte) {
	for i := 0; i < len(data); i += 4096 {
		data[i] = 1
	}
}

func readStats() runtime.MemStats {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return stats
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}


var _ = metrics.All

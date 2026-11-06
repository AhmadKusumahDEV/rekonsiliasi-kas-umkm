// Command ai-check menguji koneksi ke Gemini memakai konfigurasi yang sama
// dengan server. Jalankan setelah mengisi GEMINI_API_KEY:
//
// 	GEMINI_API_KEY=... go run ./cmd/ai-check
package main

import (
 	"context"
 	"fmt"
 	"os"
 	"time"

 	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/ai"
)

func main() {
 	if os.Getenv("GEMINI_API_KEY") == "" {
 		fmt.Println("GEMINI_API_KEY kosong; isi dulu sebelum menjalankan ai-check.")
 		os.Exit(1)
 	}
 	client := ai.NewFromEnv()
 	if client == nil {
 		fmt.Println("reviewer AI tidak aktif (periksa GEMINI_API_KEY dan AI_REVIEW_ENABLED).")
 		os.Exit(1)
 	}

 	candidates := make([]ai.Candidate, 0, 4)
 	for i := 0; i < 4; i++ {
 		candidates = append(candidates, ai.Candidate{
 			SaleID:         fmt.Sprintf("%032d", i),
 			ReceiptID:      fmt.Sprintf("%032d", 10+i),
 			Score:          0.87,
 			SaleRef:        fmt.Sprintf("INV-2%02d", 30+i),
 			ReceiptRef:     fmt.Sprintf("INV-2%02d", 30+i),
 			SaleDate:       "2026-11-08",
 			ReceiptDate:    "2026-11-08",
 			SaleAmount:     100000,
 			ReceiptAmount:  100000,
 			Channel:        "transfer",
 			ChannelEqual:   true,
 			NameSimilarity: 0.18,
 			ReferenceEqual: true,
 		})
 	}

 	start := time.Now()
 	verdicts, err := client.Review(context.Background(), candidates)
 	fmt.Printf("durasi=%s verdicts=%d err=%v\n", time.Since(start).Round(time.Millisecond), len(verdicts), err)
 	for _, verdict := range verdicts {
 		fmt.Printf("  sale=%s match=%v reason=%s\n", verdict.SaleID, verdict.Match, verdict.Reason)
 	}
 	if err != nil {
 		os.Exit(1)
 	}
}

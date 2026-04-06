package main

import (
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"os"
	"strings"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"otpdecade/proto"

	protolib "google.golang.org/protobuf/proto"
)

func main() {
	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()
	img, _, _ := image.Decode(f)
	bitmap, _ := gozxing.NewBinaryBitmapFromImage(img)
	reader := qrcode.NewQRCodeReader()
	result, _ := reader.Decode(bitmap, nil)
	uri := result.GetText()
	fmt.Println("=== Raw URI ===")
	fmt.Println(uri)
	fmt.Println()

	u, _ := url.Parse(uri)
	data := u.Query().Get("data")
	data = strings.ReplaceAll(data, " ", "+")
	fmt.Println("=== Base64 data length ===")
	fmt.Println(len(data))
	fmt.Println()

	raw, _ := base64.StdEncoding.DecodeString(data)
	fmt.Println("=== Decoded bytes length ===")
	fmt.Println(len(raw))
	fmt.Println()

	var payload proto.Payload
	protolib.Unmarshal(raw, &payload)

	fmt.Printf("=== Payload ===\n")
	fmt.Printf("Version: %d\n", payload.GetVersion())
	fmt.Printf("BatchSize: %d\n", payload.GetBatchSize())
	fmt.Printf("BatchIndex: %d\n", payload.GetBatchIndex())
	fmt.Printf("BatchID: %d\n", payload.GetBatchId())
	fmt.Printf("OtpParameters count: %d\n\n", len(payload.GetOtpParameters()))

	for i, p := range payload.GetOtpParameters() {
		secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(p.GetSecret())
		fmt.Printf("--- Entry %d ---\n", i)
		fmt.Printf("  Name: %q\n", p.GetName())
		fmt.Printf("  Issuer: %q\n", p.GetIssuer())
		fmt.Printf("  Secret: %q\n", secret)
		fmt.Printf("  Type: %v\n", p.GetType())
		fmt.Printf("  Algorithm: %v\n", p.GetAlgorithm())
		fmt.Printf("  Digits: %v\n", p.GetDigits())
		fmt.Printf("  Counter: %d\n", p.GetCounter())
		fmt.Println()
	}
}

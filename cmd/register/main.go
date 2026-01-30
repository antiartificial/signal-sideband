package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"signal-sideband/pkg/twilio"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load() // Load .env file if it exists

	signalUrl := os.Getenv("SIGNAL_URL")
	if signalUrl == "" {
		signalUrl = "http://localhost:8080"
	}
	// Sanitize URL (remove /v1/receive if present, we need base)
	signalUrl = strings.Replace(signalUrl, "/v1/receive", "", 1)
	signalUrl = strings.TrimSuffix(signalUrl, "/")

	accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	number := os.Getenv("TWILIO_PHONE_NUMBER")

	if accountSid == "" || authToken == "" || number == "" {
		log.Fatal("Missing Twilio credentials in .env")
	}

	reader := bufio.NewReader(os.Stdin)

	// Step 1: Get Captcha
	fmt.Println("Please solve a captcha at https://signalcaptchas.org/registration/generate.html")
	fmt.Println("Enter the captcha token (start with signal:captcha:...): ")
	captcha, _ := reader.ReadString('\n')
	captcha = strings.TrimSpace(captcha)

	// Step 2: Request Registration
	fmt.Printf("Registering number %s...\n", number)
	regPayload := map[string]string{
		"captcha":   captcha,
		"use_voice": "false",
	}
	regBody, _ := json.Marshal(regPayload)

	resp, err := http.Post(fmt.Sprintf("%s/v1/register/%s", signalUrl, number), "application/json", bytes.NewBuffer(regBody))
	if err != nil {
		log.Fatalf("Failed to request registration: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Registration request failed with status %d: %s", resp.StatusCode, string(body))
		log.Println("If it says 'client already registered', we will proceed to verification attempt.")
	} else {
		fmt.Println("Registration request sent. Waiting for SMS...")
	}

	// Step 3: Poll Twilio for SMS
	client := twilio.NewClient(accountSid, authToken, accountSid, number)

	var code string
	for i := 0; i < 20; i++ { // Poll for 60 seconds roughly (20 * 3s)
		fmt.Printf("Checking Twilio for SMS... (Attempt %d/20)\n", i+1)
		code, err = client.GetLatestSignalCode()
		if err == nil && code != "" {
			fmt.Printf("Found verification code: %s\n", code)
			break
		}
		time.Sleep(3 * time.Second)
	}

	if code == "" {
		log.Fatal("Could not find verification code. Please check Twilio logs manually.")
	}

	// Step 4: Verify
	fmt.Printf("Verifying with code %s...\n", code)
	verifyPayload := map[string]string{
		"code": code,
	}
	verifyBody, _ := json.Marshal(verifyPayload)

	resp, err = http.Post(fmt.Sprintf("%s/v1/register/%s/verify", signalUrl, number), "application/json", bytes.NewBuffer(verifyBody))
	if err != nil {
		log.Fatalf("Failed to verify: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Verification failed: %s", string(body))
	}

	fmt.Println("SUCCESS! Signal number registered and verified.")
}

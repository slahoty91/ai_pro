package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/slahoty/ai/types"
)

func EmbetText(client *openai.Client, text []string) ([][]float32, error) {
	response, err := client.CreateEmbeddings(context.Background(), openai.EmbeddingRequest{
		Model: openai.SmallEmbedding3,
		Input: text,
	})
	if err != nil {
		return nil, err
	}
	// fmt.Println(response.Data[0].Embedding)
	vectors := make([][]float32, len(text))
	for i, embedding := range response.Data {
		vectors[i] = embedding.Embedding
	}
	return vectors, nil
}

func ExtractStruredData(client *openai.Client, text string) (*types.ExtractedProfile, error) {
	promt := fmt.Sprintf(`
	
	You are a helpful assistant that extracts structured data from a text.
	You will be given a text and you need to extract the following information:
	1. Name
	2. Email
	3. Phone
	4. Address
	5. City
	6. State
	7. Zip
	8. Country
	9. LinkedIn
	10. Skills
	11. Most Relevant Skill
	12. Experience
	13. Education
	14. Projects
	15. Certifications
	16. Publications
	17. Companies
	18. Current Company
	19. Current Title
	20. Current Location

	Return the data in a valid JSON format.No explanations, no other text, just the JSON.

	Text: %s
	`, text)
	response, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "Extract structured JSON from documents.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: promt,
			},
		},
		Temperature: 0,
	})
	if err != nil {
		return &types.ExtractedProfile{}, err
	}

	// Clean the response content - remove markdown code blocks if present
	content := response.Choices[0].Message.Content
	content = strings.TrimSpace(content)

	// Remove markdown code blocks (```json ... ``` or ``` ... ```)
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		// Remove first line if it starts with ```
		if len(lines) > 0 && strings.HasPrefix(lines[0], "```") {
			lines = lines[1:]
		}
		// Remove last line if it's just ```
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
			lines = lines[:len(lines)-1]
		}
		content = strings.Join(lines, "\n")
	}

	var extractedProfile types.ExtractedProfile
	err = json.Unmarshal([]byte(content), &extractedProfile)
	if err != nil {
		return &types.ExtractedProfile{}, fmt.Errorf("failed to parse JSON: %w. Content: %s", err, content)
	}
	return &extractedProfile, nil
}

func ExtractPrescriptionData(client *openai.Client, text string) (*types.ExtractedPrescription, error) {
	const maxRetries = 3
	const baseDelay = 1 * time.Second

	promt := fmt.Sprintf(`
	
	You are a helpful assistant that extracts structured data from a prescription document.
	You will be given a text and you need to extract the following information:
	1. Patient Name
	2. Patient DOB
	3. Patient Address
	4. Doctor Name
	5. Doctor License (as a string, not an object)
	6. Date
	7. Prescription ID
	8. Medications (array of objects with Name, Dosage, Frequency, Duration, Instructions)
	9. Pharmacy
	10. Instructions

	Return the data in a valid JSON format. No explanations, no other text, just the JSON.

	Text: %s
	`, text)

	var lastErr error
	var lastContent string

	for attempt := 0; attempt < maxRetries; attempt++ {
		response, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "Extract structured JSON from prescription documents.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: promt,
				},
			},
			Temperature: 0,
		})
		if err != nil {
			// If API call fails and we have retries left, wait and retry
			if attempt < maxRetries-1 {
				// Exponential backoff: delay = baseDelay * 2^attempt
				delay := baseDelay * time.Duration(1<<uint(attempt+1))
				time.Sleep(delay)
				continue
			}
			return &types.ExtractedPrescription{}, err
		}

		// Clean the response content - remove markdown code blocks if present
		content := response.Choices[0].Message.Content
		content = strings.TrimSpace(content)

		// Remove markdown code blocks (```json ... ``` or ``` ... ```)
		if strings.HasPrefix(content, "```") {
			lines := strings.Split(content, "\n")
			// Remove first line if it starts with ```
			if len(lines) > 0 && strings.HasPrefix(lines[0], "```") {
				lines = lines[1:]
			}
			// Remove last line if it's just ```
			if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
				lines = lines[:len(lines)-1]
			}
			content = strings.Join(lines, "\n")
		}

		lastContent = content

		// Try to parse the JSON
		var extractedPrescription types.ExtractedPrescription
		err = json.Unmarshal([]byte(content), &extractedPrescription)

		// If parsing succeeds, return the result
		if err == nil {
			return &extractedPrescription, nil
		}

		// Store the error for the last attempt
		lastErr = err

		// If this is not the last attempt, wait and retry with exponential backoff
		if attempt < maxRetries-1 {
			// Exponential backoff: delay = baseDelay * 2^attempt
			delay := baseDelay * time.Duration(1<<uint(attempt+1))
			time.Sleep(delay)
			continue
		}
	}

	// All retries failed
	return &types.ExtractedPrescription{}, fmt.Errorf("failed to parse JSON after %d attempts: %w. Content: %s", maxRetries, lastErr, lastContent)
}

func ExtractDataWithSchema(client *openai.Client, text string, schema map[string]interface{}) (map[string]interface{}, error) {

	const maxTries = 3
	const baseDelay = 1 * time.Second
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	prompt := fmt.Sprintf(`
	You are a strict JSON extraction engine.

	You will be given:
	1. A document text
	2. A JSON schema

	Your task:
	- Populate ONLY the keys present in the schema
	- Do NOT add new keys
	- Do NOT remove keys
	- If a value is missing or unclear, use an empty string, null, or empty array (as appropriate)
	- Do NOT guess or infer values
	- Output VALID JSON ONLY (no markdown, no explanation)

	JSON Schema:
	%s

	Document Text:
	%s
	`, string(schemaBytes), text)
	var lastError error
	var lastContent string

	for attempts := 0; attempts <= maxTries; attempts++ {
		resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
			Model:       openai.GPT4oMini,
			Temperature: 0,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You return JSON only.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		})
		if err != nil {
			lastError = err
			if attempts < maxTries-1 {
				time.Sleep(baseDelay * time.Duration(1<<uint(attempts)))
				continue
			}
			return nil, err
		}

		content := strings.TrimSpace(resp.Choices[0].Message.Content)
		lastContent = content
		if strings.HasPrefix(content, "```") {
			lines := strings.Split(content, "\n")
			if len(lines) > 0 && strings.HasPrefix(lines[0], "```") {
				lines = lines[1:]
			}
			if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
				lines = lines[:len(lines)-1]
			}
			content = strings.Join(lines, "\n")
		}

		var result map[string]interface{}

		if err := json.Unmarshal([]byte(content), &result); err == nil {
			if !sameKeys(schema, result) {
				lastError = fmt.Errorf("output keys do not match schema")
			} else {
				return result, nil
			}
		} else {
			lastError = err
		}
		if attempts < maxTries-1 {
			time.Sleep(baseDelay * time.Duration(1<<uint(attempts)))
		}
	}
	return nil, fmt.Errorf(
		"failed schema extraction after %d attempts: %v. content=%s",
		maxTries,
		lastError,
		lastContent,
	)
}

func sameKeys(schema, output map[string]interface{}) bool {

	for k := range schema {
		if _, out := output[k]; !out {
			return false
		}
	}
	return true
}

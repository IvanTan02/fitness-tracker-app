package platform

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const geminiModel = "gemini-3.8-flash"

// GeminiClient calls Google's Gemini API for vision-based data extraction.
type GeminiClient struct {
	apiKey     string
	httpClient *http.Client
}

func NewGeminiClient(apiKey string) *GeminiClient {
	return &GeminiClient{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inline_data,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type geminiGenerationConfig struct {
	ResponseMimeType string `json:"response_mime_type"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// ExtractedScan holds the numeric fields Gemini read off an Evolt 360 body
// scan photo. Any field it couldn't read is left nil so the frontend can
// fall back to manual entry for just that field.
type ExtractedScan struct {
	Weight       *float64 `json:"weight"`
	BodyFat      *float64 `json:"body_fat"`
	LeanBodyMass *float64 `json:"lean_body_mass"`
	BodyFatMass  *float64 `json:"body_fat_mass"`
	SMM          *float64 `json:"smm"`
	VisceralFat  *float64 `json:"visceral_fat"`
	BMR          *float64 `json:"bmr"`
	TEE          *float64 `json:"tee"`
	LeanLeftArm  *float64 `json:"lean_left_arm"`
	LeanRightArm *float64 `json:"lean_right_arm"`
	LeanTrunk    *float64 `json:"lean_trunk"`
	LeanLeftLeg  *float64 `json:"lean_left_leg"`
	LeanRightLeg *float64 `json:"lean_right_leg"`
	FatLeftArm   *float64 `json:"fat_left_arm"`
	FatRightArm  *float64 `json:"fat_right_arm"`
	FatTrunk     *float64 `json:"fat_trunk"`
	FatLeftLeg   *float64 `json:"fat_left_leg"`
	FatRightLeg  *float64 `json:"fat_right_leg"`
}

const extractionPrompt = `You are reading a photo of an Evolt 360 body composition scan printout.
Extract the following measurements and return ONLY a JSON object with these exact keys:

- weight (kg) — from the WEIGHT field at the top of the report
- body_fat (total body fat percentage) — "TOTAL BODY FAT PERCENTAGE"
- lean_body_mass (kg) — "LEAN BODY MASS"
- body_fat_mass (kg) — "BODY FAT MASS"
- smm (skeletal muscle mass, kg) — "SKELETAL MUSCLE MASS"
- visceral_fat (visceral fat level, a unitless index number) — "VISCERAL FAT LEVEL"
- bmr (basal metabolic rate, kCal) — "BMR"
- tee (total energy expenditure, kCal) — "TEE"
- lean_left_arm (segmental lean mass, left arm, kg) — from the segmental analysis section, LEFT ARM lean mass
- lean_right_arm (segmental lean mass, right arm, kg) — RIGHT ARM lean mass
- lean_trunk (segmental lean mass, trunk, kg) — TORSO lean mass
- lean_left_leg (segmental lean mass, left leg, kg) — LEFT LEG lean mass
- lean_right_leg (segmental lean mass, right leg, kg) — RIGHT LEG lean mass
- fat_left_arm (segmental fat mass, left arm, kg) — LEFT ARM fat mass
- fat_right_arm (segmental fat mass, right arm, kg) — RIGHT ARM fat mass
- fat_trunk (segmental fat mass, trunk, kg) — TORSO fat mass
- fat_left_leg (segmental fat mass, left leg, kg) — LEFT LEG fat mass
- fat_right_leg (segmental fat mass, right leg, kg) — RIGHT LEG fat mass

Rules:
- Every value must be a number or null.
- Use null for any field you cannot read clearly or that is not present in the image.
- Do not guess or infer values that are not visibly printed.
- Return raw numbers without units in the JSON values themselves (e.g. 63.3, not "63.3 kg").`

// ExtractScan sends a body composition scan photo to Gemini and returns the
// measurements it read. imageBytes should already be validated as a
// reasonably-sized image before this is called.
func (c *GeminiClient) ExtractScan(ctx context.Context, imageBytes []byte, mimeType string) (*ExtractedScan, error) {
	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: extractionPrompt},
					{InlineData: &geminiInlineData{
						MimeType: mimeType,
						Data:     base64.StdEncoding.EncodeToString(imageBytes),
					}},
				},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			ResponseMimeType: "application/json",
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling gemini request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", geminiModel, c.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building gemini request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("calling gemini: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading gemini response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, fmt.Errorf("unmarshaling gemini response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini returned no candidates")
	}

	var extracted ExtractedScan
	rawText := geminiResp.Candidates[0].Content.Parts[0].Text
	if err := json.Unmarshal([]byte(rawText), &extracted); err != nil {
		return nil, fmt.Errorf("unmarshaling extracted scan JSON: %w", err)
	}

	return &extracted, nil
}

package main

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

func generateVideo(prompt string, bucketName string) error {
	// Configure AWS SDK
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		return fmt.Errorf("unable to load SDK config: %v", err)
	}

	// Create Bedrock Runtime client
	client := bedrockruntime.NewFromConfig(cfg)

	invocation, err := client.StartAsyncInvoke(context.TODO(), &bedrockruntime.StartAsyncInvokeInput{
		ModelId: aws.String("amazon.nova-reel-v1:0"),
		ModelInput: document.NewLazyDocument(map[string]interface{}{
			"taskType": "TEXT_VIDEO",
			"textToVideoParams": map[string]interface{}{
				"text": prompt,
			},
			"videoGenerationConfig": map[string]interface{}{
				"durationSeconds": 6,
				"fps":             24,
				"dimension":       "1280x720",
				"seed":            rand.Intn(2147483648)},
		}),
		OutputDataConfig: &types.AsyncInvokeOutputDataConfigMemberS3OutputDataConfig{
			Value: types.AsyncInvokeS3OutputDataConfig{
				S3Uri: aws.String(fmt.Sprintf("s3://%s/", bucketName)),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error invoking model: %v", err)
	}

	// 获取到的arn格式为：arn:aws:bedrock:us-east-1:900212707297:async-invoke/r6cvnsqu7kvr
	// 获取最后的r6cvnsqu7kvr作为s3的prefix,通过/分解arn得到
	arn := *invocation.InvocationArn
	prefix := arn[strings.LastIndex(arn, "/")+1:]

	// Wait for the invoke to complete
	var status types.AsyncInvokeStatus
	for {
		resp, err := client.GetAsyncInvoke(context.TODO(), &bedrockruntime.GetAsyncInvokeInput{
			InvocationArn: invocation.InvocationArn,
		})
		if err != nil {
			return fmt.Errorf("error getting invoke status: %v", err)
		}

		if resp.Status != types.AsyncInvokeStatusInProgress {
			fmt.Println("Invoke finish")
			status = resp.Status
			break
		}
		fmt.Println("Invoke status: ", resp.Status)
		time.Sleep(5 * time.Second)
	}

	if status != types.AsyncInvokeStatusCompleted {
		return fmt.Errorf("invoke failed with status: %v", status)
	}
	fmt.Printf("S3 URI: s3://%s/%s/output.mp4\n", bucketName, prefix)
	return nil
}

func main() {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	prompt := "Closeup of a large seashell in the sand. Gentle waves flow all around the shell. Sunset light. Camera zoom in very close."
	bucketName := "BUCKET_NAME" // Replace with your S3 bucket name

	if err := generateVideo(prompt, bucketName); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

package aws

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
)

//type Client struct {
//	*rekognition.Client
//}
//
//func NewRekognition(cfg aws.Config) *rekognition.Client {
//	return rekognition.NewFromConfig(cfg)
//}

func CompareFace(cfg aws.Config, source, dest []byte) (float32, error) {

	var similar float32

	cli := rekognition.NewFromConfig(cfg)

	input := &rekognition.CompareFacesInput{
		//SimilarityThreshold: aws.Float64(90.000000),
		SourceImage: &types.Image{
			Bytes: source,
		},
		TargetImage: &types.Image{
			Bytes: dest,
		},
	}

	result, err := cli.CompareFaces(context.Background(), input)
	if err == nil && len(result.FaceMatches) > 0 {
		for _, matchedFace := range result.FaceMatches {
			similar = *matchedFace.Similarity
			break
		}

	} else {
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
			fmt.Println("AWS ERROR:", errMsg)
		}
	}

	return similar, err
}

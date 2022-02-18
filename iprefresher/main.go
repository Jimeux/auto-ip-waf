package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	waf "github.com/aws/aws-sdk-go-v2/service/wafregional"
	"github.com/aws/aws-sdk-go-v2/service/wafregional/types"
)

var (
	ipSetPreviousID = aws.String(os.Getenv("IPSET_PREVIOUS_ID"))
	ipSetCurrentID  = aws.String(os.Getenv("IPSET_CURRENT_ID"))
)

var (
	waffy *waf.Client
)

type Response events.APIGatewayProxyResponse

func main() {
	cfg, _ := config.LoadDefaultConfig(context.Background())
	waffy = waf.NewFromConfig(cfg)
	lambda.Start(Handler)
}

// Handler updates the previous and current IP sets.
// - previous - old IPs deleted and replaced with IPs from current
// - current - old IPs transferred to previous and replace with new IPs from the server
func Handler(ctx context.Context) (Response, error) {
	// fetch new IPs from server here
	newIPs := []string{"220.224.0.1/32"}

	// fetch IP sets
	previous, err := waffy.GetIPSet(ctx, &waf.GetIPSetInput{IPSetId: ipSetPreviousID})
	if err != nil {
		return Response{StatusCode: http.StatusInternalServerError}, err
	}
	current, err := waffy.GetIPSet(ctx, &waf.GetIPSetInput{IPSetId: ipSetCurrentID})
	if err != nil {
		return Response{StatusCode: http.StatusInternalServerError}, err
	}

	// update previous
	if len(previous.IPSet.IPSetDescriptors) != 0 || len(current.IPSet.IPSetDescriptors) != 0 {
		prevToken, err := waffy.GetChangeToken(ctx, &waf.GetChangeTokenInput{})
		if err != nil {
			return Response{StatusCode: 500}, err
		}
		prevParams := buildUpdateParamsForPrevious(previous, current, prevToken.ChangeToken)
		if _, err := waffy.UpdateIPSet(ctx, prevParams); err != nil {
			return Response{StatusCode: 500}, err
		}
	}

	// update current
	currToken, err := waffy.GetChangeToken(ctx, &waf.GetChangeTokenInput{})
	if err != nil {
		return Response{StatusCode: 500}, err
	}
	if _, err := waffy.UpdateIPSet(ctx, buildUpdateParamsForCurrent(current, newIPs, currToken.ChangeToken)); err != nil {
		return Response{StatusCode: 500}, err
	}

	var buf bytes.Buffer
	body, err := json.Marshal(map[string]interface{}{
		"message": fmt.Sprintf("IP sets updated WAFully! [Prev: %s], [Curr: %s]",
			*previous.IPSet.Name, *current.IPSet.Name),
	})
	if err != nil {
		return Response{StatusCode: 404}, err
	}
	json.HTMLEscape(&buf, body)

	return Response{
		StatusCode: 200,
		Body:       buf.String(),
		Headers:    map[string]string{"Content-Type": "application/json"},
	}, nil
}

func buildUpdateParamsForPrevious(previous, current *waf.GetIPSetOutput, token *string) *waf.UpdateIPSetInput {
	params := &waf.UpdateIPSetInput{
		ChangeToken: token,
		IPSetId:     previous.IPSet.IPSetId,
		Updates:     []types.IPSetUpdate{},
	}

	// delete previous IPs
	for _, desc := range previous.IPSet.IPSetDescriptors {
		params.Updates = append(params.Updates, types.IPSetUpdate{
			Action:          types.ChangeActionDelete,
			IPSetDescriptor: &desc,
		})
	}
	// replace previous with current
	for _, desc := range current.IPSet.IPSetDescriptors {
		params.Updates = append(params.Updates, types.IPSetUpdate{
			Action:          types.ChangeActionInsert,
			IPSetDescriptor: &desc,
		})
	}
	return params
}

func buildUpdateParamsForCurrent(current *waf.GetIPSetOutput, newIPs []string, token *string) *waf.UpdateIPSetInput {
	params := &waf.UpdateIPSetInput{
		ChangeToken: token,
		IPSetId:     current.IPSet.IPSetId,
		Updates:     []types.IPSetUpdate{},
	}
	// delete current IPs
	for _, desc := range current.IPSet.IPSetDescriptors {
		params.Updates = append(params.Updates, types.IPSetUpdate{
			Action:          types.ChangeActionDelete,
			IPSetDescriptor: &desc,
		})
	}
	// replace with newly acquired IPs
	for _, ip := range newIPs {
		params.Updates = append(params.Updates, types.IPSetUpdate{
			Action: types.ChangeActionInsert,
			IPSetDescriptor: &types.IPSetDescriptor{
				Type:  types.IPSetDescriptorTypeIpv4,
				Value: &ip,
			},
		})
	}
	return params
}

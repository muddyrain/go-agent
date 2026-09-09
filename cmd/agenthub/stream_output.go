package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/cloudwego/eino/schema"
)

func writeAssistantStream(
	writer io.Writer,
	stream *schema.StreamReader[*schema.Message],
) (int, error) {
	defer stream.Close()

	chunks := 0

	for {
		chunk, err := stream.Recv()

		if errors.Is(err, io.EOF) {
			return chunks, nil
		}

		if err != nil {
			return chunks, fmt.Errorf("receive assistant stream: %w", err)
		}

		if chunk == nil {
			return chunks, fmt.Errorf("receive assistant stream: nil chunk")
		}

		chunks++

		if _, err := fmt.Fprint(writer, chunk.Content); err != nil {
			return chunks, fmt.Errorf("write assistant stream: %w", err)
		}
	}
}

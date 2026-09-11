package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/cloudwego/eino/schema"
)

// writeAssistantStream 只负责消费最终 Assistant 文本流并写入终端。
// 它拥有传入的 StreamReader，因此必须在函数退出时关闭；io.EOF 表示
// 正常读取完成，其他错误表示本轮只得到不完整输出，应交给 run 决定是否回滚历史。
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

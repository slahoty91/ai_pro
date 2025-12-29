package utils

import (
	"strings"
)

func ChunkText(text string, chunkSize int) []string {
	var chunks []string
	// for len(text) > chunkSize {
	// 	chunks = append(chunks, text[:chunkSize])
	// 	text = text[chunkSize:]
	// }
	// chunks = append(chunks, text)
	for len(text) > chunkSize {
		end := chunkSize
		for i := end; i > 0 && i < len(text); i-- {
			if text[i] == ' ' {
				end = i
				break
			}
		}
		chunks = append(chunks, strings.TrimSpace(text[:end]))
		// fmt.Println(chunks, "chunksss in for looppppp")
		text = strings.TrimSpace(text[end:])
	}
	if len(text) > 0 {
		chunks = append(chunks, text)
	}
	// fmt.Println(len(chunks), "length of chunkss", chunks)
	return chunks
}

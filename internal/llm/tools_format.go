package llm

// ConvertToolsForProvider converts tool definitions to the format expected by a provider
func ConvertToolsForProvider(tools []ToolDefinition, format ToolFormat) []map[string]interface{} {
	if len(tools) == 0 {
		return nil
	}

	result := make([]map[string]interface{}, 0, len(tools))

	for _, tool := range tools {
		switch format {
		case ToolFormatAnthropic:
			// Anthropic format: { name, description, input_schema }
			result = append(result, map[string]interface{}{
				"name":         tool.Name,
				"description":  tool.Description,
				"input_schema": tool.InputSchema,
			})

		case ToolFormatOpenAI:
			// OpenAI format: { type: "function", function: { name, description, parameters } }
			result = append(result, map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        tool.Name,
					"description": tool.Description,
					"parameters":  tool.InputSchema,
				},
			})

		case ToolFormatGoogle:
			// Google format: { functionDeclarations: [{ name, description, parameters }] }
			// For Google, we return a single object with all functions
			result = append(result, map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.InputSchema,
			})
		}
	}

	return result
}

// WrapToolsForGoogle wraps tools in Google's expected format
func WrapToolsForGoogle(tools []map[string]interface{}) []map[string]interface{} {
	if len(tools) == 0 {
		return nil
	}
	return []map[string]interface{}{
		{
			"functionDeclarations": tools,
		},
	}
}

// BuildRequestWithTools adds tools to a request body based on provider format
func BuildRequestWithTools(body map[string]interface{}, tools []ToolDefinition, format ToolFormat) {
	if len(tools) == 0 {
		return
	}

	converted := ConvertToolsForProvider(tools, format)

	switch format {
	case ToolFormatAnthropic:
		body["tools"] = converted
	case ToolFormatOpenAI:
		body["tools"] = converted
	case ToolFormatGoogle:
		body["tools"] = WrapToolsForGoogle(converted)
	}
}

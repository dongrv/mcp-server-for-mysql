// Package schema provides JSON Schema definitions for all MCP tools.
package schema

import (
	"encoding/json"
	"fmt"
	"strings"
)

// GenerateMarkdownDocumentation generates markdown documentation for all tools.
func GenerateMarkdownDocumentation() string {
	schemas := GetToolSchemas()
	var builder strings.Builder

	builder.WriteString("# MySQL MCP Tools Schema Documentation\n\n")
	builder.WriteString("## Overview\n\n")
	builder.WriteString("This document describes the JSON Schema for all MySQL MCP tools.\n\n")
	builder.WriteString("## Tool Categories\n\n")

	// Group tools by category
	categories := map[string][]string{
		"Query & Execution": {
			"mysql_query",
			"mysql_execute",
		},
		"Transaction Management": {
			"mysql_begin_transaction",
			"mysql_commit_transaction",
			"mysql_rollback_transaction",
		},
		"Table Operations": {
			"mysql_list_tables",
			"mysql_describe_table",
			"mysql_create_table",
			"mysql_drop_table",
			"mysql_rename_table",
			"mysql_copy_table",
			"mysql_copy_table_structure",
		},
		"Column Operations": {
			"mysql_add_columns",
			"mysql_drop_columns",
			"mysql_modify_columns",
		},
		"Index Operations": {
			"mysql_create_index",
			"mysql_drop_index",
			"mysql_list_indexes",
		},
		"Database Management": {
			"mysql_migrate",
			"mysql_pool_status",
		},
	}

	// Generate category overview
	for category, toolNames := range categories {
		builder.WriteString(fmt.Sprintf("### %s\n\n", category))
		for _, toolName := range toolNames {
			if schema, exists := schemas[toolName]; exists {
				builder.WriteString(fmt.Sprintf("- **%s**: %s\n", schema.Name, schema.Description))
			}
		}
		builder.WriteString("\n")
	}

	// Generate detailed schema documentation
	builder.WriteString("## Detailed Schema Reference\n\n")

	for _, category := range []string{
		"Query & Execution",
		"Transaction Management",
		"Table Operations",
		"Column Operations",
		"Index Operations",
		"Database Management",
	} {
		builder.WriteString(fmt.Sprintf("### %s\n\n", category))

		for _, toolName := range categories[category] {
			schema, exists := schemas[toolName]
			if !exists {
				continue
			}

			builder.WriteString(fmt.Sprintf("#### %s\n\n", schema.Name))
			builder.WriteString(fmt.Sprintf("**Description**: %s\n\n", schema.Description))

			// Parse and format schema
			var schemaObj map[string]interface{}
			if err := json.Unmarshal(schema.InputSchema, &schemaObj); err == nil {
				builder.WriteString("**Input Schema**:\n\n")
				builder.WriteString("```json\n")
				formatted, _ := json.MarshalIndent(schemaObj, "", "  ")
				builder.WriteString(string(formatted))
				builder.WriteString("\n```\n\n")

				// Extract required fields
				if required, ok := schemaObj["required"].([]interface{}); ok && len(required) > 0 {
					builder.WriteString("**Required Fields**:\n\n")
					for _, field := range required {
						builder.WriteString(fmt.Sprintf("- `%s`\n", field))
					}
					builder.WriteString("\n")
				}

				// Extract properties
				if properties, ok := schemaObj["properties"].(map[string]interface{}); ok && len(properties) > 0 {
					builder.WriteString("**Properties**:\n\n")
					for propName, propValue := range properties {
						propMap, ok := propValue.(map[string]interface{})
						if !ok {
							continue
						}

						builder.WriteString(fmt.Sprintf("- `%s`:\n", propName))
						if desc, ok := propMap["description"].(string); ok && desc != "" {
							builder.WriteString(fmt.Sprintf("  - **Description**: %s\n", desc))
						}
						if propType, ok := propMap["type"].(string); ok && propType != "" {
							builder.WriteString(fmt.Sprintf("  - **Type**: `%s`\n", propType))
						}
						if enum, ok := propMap["enum"].([]interface{}); ok && len(enum) > 0 {
							builder.WriteString("  - **Allowed Values**: ")
							enumStrs := make([]string, len(enum))
							for i, e := range enum {
								enumStrs[i] = fmt.Sprintf("`%v`", e)
							}
							builder.WriteString(strings.Join(enumStrs, ", "))
							builder.WriteString("\n")
						}
						builder.WriteString("\n")
					}
				}
			}

			builder.WriteString("---\n\n")
		}
	}

	// Add usage examples
	builder.WriteString("## Usage Examples\n\n")

	builder.WriteString("### Example 1: Query Data\n\n")
	builder.WriteString("```json\n")
	builder.WriteString(`{
  "query": "SELECT * FROM users WHERE status = ?",
  "parameters": ["active"]
}`)
	builder.WriteString("\n```\n\n")

	builder.WriteString("### Example 2: Create Table\n\n")
	builder.WriteString("```json\n")
	builder.WriteString(`{
  "table_name": "products",
  "columns": "id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100) NOT NULL, price DECIMAL(10,2) NOT NULL, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP"
}`)
	builder.WriteString("\n```\n\n")

	builder.WriteString("### Example 3: Add Columns\n\n")
	builder.WriteString("```json\n")
	builder.WriteString(`{
  "table_name": "users",
  "columns": [
    {
      "name": "email",
      "type": "VARCHAR(255)",
      "nullable": false,
      "default_value": ""
    },
    {
      "name": "phone",
      "type": "VARCHAR(20)",
      "nullable": true,
      "after_column": "email"
    }
  ]
}`)
	builder.WriteString("\n```\n\n")

	builder.WriteString("### Example 4: Create Index\n\n")
	builder.WriteString("```json\n")
	builder.WriteString(`{
  "table_name": "orders",
  "index_name": "idx_customer_date",
  "columns": ["customer_id", "order_date"],
  "index_type": "INDEX"
}`)
	builder.WriteString("\n```\n\n")

	return builder.String()
}

// GenerateJSONSchemaDocument generates a complete JSON Schema document for all tools.
func GenerateJSONSchemaDocument() (string, error) {
	schemas := GetToolSchemas()

	// Create a map of all schemas
	schemaMap := make(map[string]interface{})
	for name, schema := range schemas {
		var schemaObj map[string]interface{}
		if err := json.Unmarshal(schema.InputSchema, &schemaObj); err == nil {
			schemaMap[name] = map[string]interface{}{
				"name":        schema.Name,
				"description": schema.Description,
				"schema":      schemaObj,
			}
		}
	}

	// Create the complete document
	document := map[string]interface{}{
		"title":       "MySQL MCP Tools JSON Schema",
		"description": "Complete JSON Schema definitions for all MySQL MCP tools",
		"version":     "1.0.0",
		"tools":       schemaMap,
		"timestamp":   "2024-01-01T00:00:00Z",
	}

	// Marshal with indentation
	jsonBytes, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON schema document: %w", err)
	}

	return string(jsonBytes), nil
}

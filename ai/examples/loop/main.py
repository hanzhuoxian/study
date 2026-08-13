from openai import OpenAI

client = OpenAI(base_url="http://localhost:11434/v1", api_key="ollama")

# ── Tool definitions ──
tools = [
    {
        "type": "function",
        "function": {
            "name": "get_current_time",
            "description": "Get the current date and time in a specific timezone",
            "parameters": {
                "type": "object",
                "properties": {
                    "timezone": {
                        "type": "string",
                        "description": "Timezone name, e.g. America/Vancouver",
                    }
                },
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "get_weather",
            "description": "Get the current weather for a specific city",
            "parameters": {
                "type": "object",
                "properties": {
                    "city": {"type": "string", "description": "City name"},
                    "unit": {"type": "string", "enum": ["celsius", "fahrenheit"]},
                },
            },
        },
    },
]


# ── Tool execution function (stub with canned results; a real implementation
# must parse the JSON `arguments` and call actual APIs) ──
def execute_tool(name, arguments):
    if name == "get_current_time":
        return '{"datetime": "2025-09-13T05:18:47", "day_of_week": "Saturday"}'
    elif name == "get_weather":
        return '{"temperature": 13.2, "unit": "celsius", "conditions": "clear", "humidity": 93}'


# ── Initial message list ──
messages = [
    {
        "role": "system",
        "content": "You are a helpful assistant. Use tools to get real-time information when needed.",
    },
    {"role": "user", "content": "What's the current time and weather in Vancouver?"},
]


def main():
    # ── Agent core loop ──
    # Production code needs a max_iterations cap here: as discussed later in
    # this chapter, Agents can get stuck repeating the same tool calls forever
    iteration = 0
    while True:
        iteration += 1
        print(f"\n=== Iteration {iteration}: calling model ===")
        response = client.chat.completions.create(
            model="qwen3:0.6b", messages=messages, tools=tools
        )
        assistant_message = response.choices[0].message

        # Append model's response to message list (whether text or tool calls)
        messages.append(assistant_message)

        # If no tool calls requested, the model has produced its final response
        if not assistant_message.tool_calls:
            print("--- Final response (no tool calls requested) ---")
            print(assistant_message)
            break

        # Execute each tool requested by the model, append results to message list
        for tool_call in assistant_message.tool_calls:
            print(
                f"--- Tool call requested: {tool_call.function.name}"
                f"({tool_call.function.arguments}) ---"
            )
            result = execute_tool(tool_call.function.name, tool_call.function.arguments)
            print(f"--- Tool result: {result} ---")
            messages.append(
                {
                    "role": "tool",
                    "tool_call_id": tool_call.id,
                    "content": result,
                }
            )
        # Return to top of loop, call model again with updated message list


if __name__ == "__main__":
    main()

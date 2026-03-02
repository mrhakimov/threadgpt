import urllib.request
import json

api_key = input("Enter your OpenAI API key: ").strip()

payload = json.dumps({
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "hello"}]
}).encode("utf-8")

req = urllib.request.Request(
    "https://api.openai.com/v1/chat/completions",
    data=payload,
    headers={
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json"
    }
)

with urllib.request.urlopen(req) as response:
    result = json.loads(response.read())

print(result["choices"][0]["message"]["content"])

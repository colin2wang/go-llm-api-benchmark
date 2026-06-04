# API Example

```shell
	curl -X POST 'https://llm.huzhou.gov.cn/mmttapi/v1/chat-messages' \
	--header 'Authorization: Bearer {api_key}' \
	--header 'Content-Type: application/json' \
	--header 'x-motu-deptid: d-825f4a25ca0346fea708' \
	--header 'x-motu-projectId: p-7a478b2d9c2f416ab42f' \
	--data-raw '{
	    "inputs": {},
	    "query": "What are the specs of the iPhone 13 Pro Max?",
	    "response_mode": "streaming",
	    "conversation_id": "",
	    "user": "abc-123",
	    "files": [
	      {
	        "type": "image",
	        "transfer_method": "remote_url",
	        "url": "https://xxx.cn/abc.png"
	      }
	    ]
}'
```

> Create a config.yaml file, and i will copy the value of {api_key} to the file.
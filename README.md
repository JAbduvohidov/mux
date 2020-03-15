_**`single`**_

[https://example.com/{example}] - like queries can be gathered by **request.Context().Value("example")**

Also woks with **_`multiple`_**
[https://example.com/{example}/{example2}]
request.Context().Value("example")
request.Context().Value("example2")

Special thanks to @coursar and @AlisherFozilov
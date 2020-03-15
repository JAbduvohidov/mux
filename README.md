#### Works with **`single`** value:

[https://example.com/{value1}] - values can be gathered by **request.Context().Value("example")**

#### And woks with **`multiple`** values:

[https://example.com/{value1}/{value2}]
**_request.Context().Value("value1")_** - gets the first value

**_request.Context().Value("value2")_** - gets the second value

Special thanks to **@coursar** and **@AlisherFozilov**
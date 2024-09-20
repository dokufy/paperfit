import "./common/operation.tsp";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace Paperfit;

model DocumentFileInput {
    input: Input;
    files: bytes[];
}

@route("/pdf/from/html/file")
@tag("Generate from HTML")
interface FileRoutes {
    @post
    @operationId("createDocumentFromFile")
    @summary("Create document from file")
    /**
     * Takes static HTML project files consisting of 
	 * at least a index.html file and optional variables
	 * to be combined during PDF generation.
	 * The variables are available as global variables to
	 * js scripts defined in the project.
	 * Variables can be validated with a json_schema.json
	 * file present in the archive.
	 * No caching is performed for this request.
     */
    FromFile(
        @header contentType: "multipart/form-data",
        @body input: DocumentFileInput,
    ): bytes | UnauthorizedError | SharedError;
}
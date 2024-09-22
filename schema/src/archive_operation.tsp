import "./common/operation.tsp";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace Paperfit;

model DocumentArchiveInput {
    rootPath?: string;
    file: bytes;
    input?: Input;
}

@route("/pdf/from/archive/file")
@tag("Generate from archive")
interface ArchiveRoutes {
    @post
    @operationId("createDocumentFromArchive")
    @summary("Create document from archive")
    /**
    * Takes an archive containing a static HTML project
    * consisting of at least a <b>index.html</b> file and optional variables
    * to be combined during PDF generation.<br />
    * The variables are available as global variables to
    * js scripts defined in the project.<br />
    * Variables can be validated with a <b>json_schema.json</b>
    * file contained in the archive.<br />
    * The generated PDF is cached to improve performance,
    * but the cache is cleared when the archive field name or variables change.<br />
    */
    FromArchive(
        @header contentType: "multipart/form-data",
        @body input: DocumentArchiveInput,
    ): bytes | UnauthorizedError | SharedError;
}

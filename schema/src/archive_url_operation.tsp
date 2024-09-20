import "./common/operation.tsp";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace Paperfit;

model DocumentArchiveUrlInput {
    additionalHeaders?: Record<string>;
    input: Input;
    url: string;
}

@route("/pdf/from/archive/url")
@tag("Generate from archive url")
interface ArchiveUrlRoutes {
    @post
    @operationId("createDocumentFromArchiveUrl")
    @summary("Create document from archive url")
    /**
     * Fetches an archive at the given url containing a static HTML project
	 * consisting of at least a <b>index.html</b> file and optional variables
	 * to be combined during PDF generation.<br />
	 * You can pass additional <i>headers</i> that will be used when fetching the archive,
	 * for example, in case the request needs to authenticate to download.<br />
	 * The variables are available as global variables to
	 * js scripts defined in the project.<br />
	 * Variables can be validated with a <b>json_schema.json</b>
	 * file contained in the archive.<br />
	 * The generated PDF and archive are cached to improve performance,
     * but the cache is cleared when the archive url, variables or headers change.
     */
    FromArchiveUrl(
        @body input: DocumentArchiveUrlInput,
    ): bytes | UnauthorizedError | SharedError;
}
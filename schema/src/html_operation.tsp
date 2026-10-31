import "./common/operation.tsp";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace Paperfit;

model DocumentHtmlInput {
    html: string;
    assets?: Record<string>;
    jsonSchema?: unknown;
    input?: Input;
    rootPath?: string;
    options?: {
        landscape?: boolean;
        waitDelay?: string;
        waitForExpression?: string;
        paperWidth?: string;
        paperHeight?: string;
        marginTop?: string;
        marginBottom?: string;
        marginLeft?: string;
        marginRight?: string;
        preferCssPageSize?: boolean;
        printBackground?: boolean;
    };
}

@route("/pdf/from/html")
@tag("Generate from HTML")
interface HtmlRoutes {
    @post
    @operationId("createDocumentFromHtml")
    @summary("Create document from HTML")
    FromHtml(
        @body input: DocumentHtmlInput,
    ): bytes | UnauthorizedError | SharedError;
}

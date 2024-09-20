import "@typespec/openapi";
import "@typespec/openapi3";

import "./application.tsp";
import "./archive_operation.tsp";
import "./archive_url_operation.tsp";
import "./file_operation.tsp";

using TypeSpec.Http;

@service({
    title: "paperfit",
})
@server("http://localhost:3000", "Local Server")
@useAuth(BearerAuth)
namespace Paperfit;

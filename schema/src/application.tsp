import "@typespec/http";
import "@typespec/rest";

import "./common/error.tsp";

namespace Paperfit;

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

@error
@doc("Application not ready")
model ApplicationNotReadyError {
    @statusCode
    statusCode: 500;
}

@route("/application")
@tag("Application")
interface ApplicationRoutes {
    @get
    @route("/healthy")
    @operationId("healthy")
    @useAuth(NoAuth)
    @summary("Check if the application is running")
    Healthy(): boolean | ApplicationNotReadyError | SharedError;
}

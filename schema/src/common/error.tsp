import "@typespec/http";

using TypeSpec.Http;

@error
@doc("Rate limit exceeded")
model RateLimitError {
    @statusCode
    statusCode: 429;

    message: string;
}

@error
@doc("Bad request")
model ValidationError {
    @statusCode
    statusCode: 400;

    message: string;
}

@error
@doc("Unauthorized error")
model UnauthorizedError {
    @statusCode
    statusCode: 401;

    message: string;
}

@error
@doc("Unknown error")
model UnknownError {
    @statusCode
    statusCode: 500;

    message: string;
}

alias SharedError = RateLimitError | ValidationError | UnknownError;

// sleep for x ms
export function sleep(ms: number) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

// get baseUrl sent from server rendered index template
export function baseUrl() {
    let baseUrl = ""
    if (window.APP.baseUrl) {
        if (window.APP.baseUrl === "/") {
            baseUrl = "/"
        } else if (window.APP.baseUrl === "{{.BaseUrl}}") {
            baseUrl = "/"
        } else if (window.APP.baseUrl === "/autobrr/") {
            baseUrl = "/autobrr/"
        } else {
            baseUrl = window.APP.baseUrl
        }
    }

    return baseUrl
}

// get sseBaseUrl for SSE
export function sseBaseUrl() {
    let {protocol, origin} = window.location

    let env = process.env.NODE_ENV
    if (env === "development") {
        return `http://localhost:8989/`
    }

    return `${protocol}//${origin}${baseUrl()}`
}

export function buildPath(...args: string[]): string {
    const [first] = args;
    const firstTrimmed = first.trim();
    const result = args
        .map((part) => part.trim())
        .map((part, i) => {
            if (i === 0) {
                return part.replace(/[/]*$/g, '');
            } else {
                return part.replace(/(^[/]*|[/]*$)/g, '');
            }
        })
        .filter((x) => x.length)
        .join('/');

    return firstTrimmed === '/' ? `/${result}` : result;
}

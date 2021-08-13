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
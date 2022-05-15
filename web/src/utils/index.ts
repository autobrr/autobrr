import { formatDistanceToNowStrict, formatISO9075 } from "date-fns";

// sleep for x ms
export function sleep(ms: number) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

// get baseUrl sent from server rendered index template
export function baseUrl() {
    let baseUrl = "";
    if (window.APP.baseUrl) {
        if (window.APP.baseUrl === "/") {
            baseUrl = "/";
        } else if (window.APP.baseUrl === "{{.BaseUrl}}") {
            baseUrl = "/";
        } else if (window.APP.baseUrl === "/autobrr/") {
            baseUrl = "/autobrr/";
        } else {
            baseUrl = window.APP.baseUrl;
        }
    }

    return baseUrl;
}

// get sseBaseUrl for SSE
export function sseBaseUrl() {
    if (process.env.NODE_ENV === "development")
        return "http://localhost:7474/";

    return `${window.location.origin}${baseUrl()}`;
}

export function classNames(...classes: string[]) {
    return classes.filter(Boolean).join(" ");
}

// column widths for inputs etc
export type COL_WIDTHS = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12;

// simplify date
export function simplifyDate(date: string) {
    if (date !== "0001-01-01T00:00:00Z") {
        return formatISO9075(new Date(date));
    }
    return "n/a";
}

// if empty date show as n/a
export function IsEmptyDate(date: string) {
    if (date !== "0001-01-01T00:00:00Z") {
        return formatDistanceToNowStrict(
            new Date(date),
            { addSuffix: true }
        );
    }
    return "n/a"
}

export function slugify(str: string) {
    return str
        .normalize('NFKD')
        .toLowerCase()
        .replace(/[^\w\s-]/g, '')
        .trim()
        .replace(/[-\s]+/g, '-');
}

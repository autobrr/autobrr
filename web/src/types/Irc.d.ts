interface IrcNetwork {
    id: number;
    name: string;
    enabled: boolean;
    addr: string;
    server: string;
    port: string;
    nick: string;
    username: string;
    realname: string;
    pass: string;
    connected: boolean;
    connected_since: string;
    tls: boolean;
    nickserv: {
        account: string;
    }
    channels: IrcNetworkChannel[];
}

interface IrcNetworkChannel {
    id: number;
    enabled: boolean;
    name: string;
    password: string;
    detached: boolean;
    monitoring: boolean;
    monitoring_since: string;
    last_announce: string;
}

interface NickServ {
    account: string;
    password: string;
}

interface Network {
    id?: number;
    name: string;
    enabled: boolean;
    server: string;
    port: number;
    tls: boolean;
    invite_command: string;
    nickserv: {
        account: string;
        password: string;
    }
    channels: Channel[];
    settings: object;
}

interface Channel {
    name: string;
    password: string;
}

interface SASL {
    mechanism: string;
    plain: {
        username: string;
        password: string;
    }
}

interface Config {
    host: string;
    port: number;
    log_level: string;
    log_path: string;
    base_url: string;
    version: string;
    commit: string;
    date: string;
}
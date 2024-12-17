/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { formatISO9075 } from "date-fns";
import * as CONST from "./_const";

type SubPrimitive = string | number | boolean | symbol | undefined;
type Primitive = SubPrimitive | SubPrimitive[];

type ValueObject = Record<string, Primitive>;

class ParserFilter {
  public values: ValueObject = {};
  public warnings: string[] = [];

  public name: string;

  constructor(name: string) {
    this.name = name;
    this.values = {
      name: `${name} - autodl-irssi import (${formatISO9075(Date.now())})`
    };
  }

  public OnParseLine(key: string, value: string) {
    if (key in CONST.FILTER_SUBSTITUTION_MAP) {
      key = CONST.FILTER_SUBSTITUTION_MAP[key];
    }

    switch (key) {
    case "log_score": {
      // In this case we need to set 2 properties in autobrr instead of only 1
      this.values["log"] = true;

      // log_score is an integer
      const delim = value.indexOf("-");
      if (delim !== -1) {
        value = value.slice(0, delim);
      }
      break;
    }
    case "max_downloads_unit": {
      value = value.toUpperCase();
      break;
    }
    default:
      break;
    }

    if (key in CONST.FILTER_FIELDS) {
      switch (CONST.FILTER_FIELDS[key]) {
      case "number": {
        const parsedNum = parseFloat(value);
        this.values[key] = parsedNum;

        if (isNaN(parsedNum)) {
          this.warnings.push(
            `[Filter=${this.name}] Failed to convert field '${key}' to a number. Got value: '${value}'`
          );
        }
        break;
      }
      case "boolean": {
        this.values[key] = value.toLowerCase() === "true";
        break;
      }
      default:
        this.values[key] = value;
        break;
      }
    } else {
      this.values[key] = value;
    }
  }

  private FixupMatchLogic(fieldName: string) {
    const logicAnyField = `${fieldName}_any`;
    if (logicAnyField in this.values && fieldName in this.values) {
      this.values[`${fieldName}_match_logic`] = this.values[logicAnyField] ? "ANY" : "ALL";
    }

    delete this.values[logicAnyField];
  }

  public FixupValues() {
    // Force-disable this filter
    this.values["enabled"] = false;

    // Convert into string arrays if necessary
    for (const key of Object.keys(this.values)) {
      // If key is not in FILTER_FIELDS, skip...
      if (!(key in CONST.FILTER_FIELDS)) {
        continue;
      }

      const keyType = CONST.FILTER_FIELDS[key];
      if (!keyType.endsWith("string")) {
        continue;
      }

      if (Array.isArray(this.values[key])) {
        continue;
      }

      // Split a string by ',' and create an array out of it
      const entries = (this.values[key] as string)
        .split(",")
        .map((v) => v.trim());

      this.values[key] = keyType === "string" ? entries.join(",") : entries;
    }

    // Add missing []string fields
    for (const [fieldName, fieldType] of Object.entries(CONST.FILTER_FIELDS)) {
      if (fieldName in this.values) {
        continue;
      }

      if (fieldType === "[]string") {
        this.values[fieldName] = [];
      }
    }

    this.FixupMatchLogic("tags");
    this.FixupMatchLogic("except_tags");
  }
}

class ParserIrcNetwork {
  public values: ValueObject = {};
  public warnings: string[] = [];

  private serverName: string;

  constructor(serverName: string) {
    this.serverName = serverName;
    this.values = {
      "name": serverName,
      "server": serverName,
      "channels": []
    };
  }

  public OnParseLine(key: string, value: string) {
    this.warnings.push(
      `[IrcNetwork=${this.serverName}] Autobrr currently doesn't support import of field '${key}' (value: ${value})`
    );
    /*if (key in CONST.IRC_SUBSTITUTION_MAP) {
      key = CONST.IRC_SUBSTITUTION_MAP[key];
    }

    if (key in CONST.IRC_FIELDS) {
      switch (CONST.IRC_FIELDS[key]) {
      case "number": {
        const parsedNum = parseFloat(value);
        this.values[key] = parsedNum;

        if (isNaN(parsedNum)) {
          this.warnings.push(
            `[IrcNetwork=${this.serverName}] Failed to convert field '${key}' to a number. Got value: '${value}'`
          );
        }
        break;
      }
      case "boolean": {
        this.values[key] = value.toLowerCase() === "true";
        break;
      }
      default: {
        break;
      }
      }
    } else {
      this.values[key] = value;
    }*/
  }

  public FixupValues() {
    this.values["enabled"] = false;
  }

  public GetChannels() {
    return this.values["channels"];
  }
}

class ParserIrcChannel {
  public values: ValueObject = {};
  public warnings: string[] = [];

  public serverName: string;

  constructor(serverName: string) {
    this.serverName = serverName;
  }

  public OnParseLine(key: string, value: string) {
    // TODO: autobrr doesn't respect invite-command
    // if (["name", "password"].includes(key))
    //   this.values[key] = value;

    this.warnings.push(
      `[IrcChannel=${this.serverName}] Autobrr currently doesn't support import of field '${key}' (value: ${value})`
    );
  }

  public FixupValues() {
    this.values["enabled"] = false;
  }
}

// erm.. todo?
// const TRACKER = "tracker" as const;
// const OPTIONS = "options" as const;

// *cough* later dude, trust me *cough*
const FILTER = "filter" as const;
const SERVER = "server" as const;
const CHANNEL = "channel" as const;

export class AutodlIrssiConfigParser {
  // Temporary storage objects
  private releaseFilter?: ParserFilter = undefined;
  private ircNetwork?: ParserIrcNetwork = undefined;
  private ircChannel?: ParserIrcChannel = undefined;

  // Where we'll keep our parsed objects
  public releaseFilters: ParserFilter[] = [];
  public ircNetworks: ParserIrcNetwork[] = [];
  public ircChannels: ParserIrcChannel[] = [];

  private regexHeader: RegExp = new RegExp(/\[([^\s]*)\s?(.*?)?]/);
  private regexKeyValue: RegExp = new RegExp(/([^\s]*)\s?=\s?(.*)/);

  private sectionName: string = "";

  // Save content we've parsed so far
  private Save() {
    if (this.releaseFilter !== undefined) {
      this.releaseFilter.FixupValues();
      this.releaseFilters.push(this.releaseFilter);
    } else if (this.ircNetwork !== undefined) {
      this.ircNetwork.FixupValues();
      this.ircNetworks.push(this.ircNetwork);
    } else if (this.ircChannel !== undefined) {
      this.ircChannel.FixupValues();
      this.ircChannels.push(this.ircChannel);
    }

    this.releaseFilter = undefined;
    this.ircNetwork = undefined;
    this.ircChannel = undefined;
  }

  private GetHeader(line: string): boolean {
    const match = line.match(this.regexHeader);
    if (!match) {
      return false;
    }

    this.Save();
    this.sectionName = match[1];

    const rightLeftover = match[2];
    if (!rightLeftover) {
      return true;
    }

    switch (match[1]) {
    case FILTER: {
      this.releaseFilter = new ParserFilter(rightLeftover);
      break;
    }
    case SERVER: {
      this.ircNetwork = new ParserIrcNetwork(rightLeftover);
      break;
    }
    case CHANNEL: {
      this.ircChannel = new ParserIrcChannel(rightLeftover);
      break;
      }
    default: {
      break;
    }
    }

    return true;
  }

  public GetWarnings() {
    return this.releaseFilters.flatMap((filter) => filter.warnings);
  }

  public Parse(content: string) {
    content.split("\n").forEach((line) => {
      line = line.trim();

      if (!line.length) {
        return;
      }

      // Header was parsed, go further
      if (this.GetHeader(line)) {
        return;
      }

      if (!this.sectionName.length) {
        return;
      }

      const match = line.match(this.regexKeyValue);
      if (!match) {
        return;
      }

      const key = match[1].replaceAll("-", "_").trim();
      const value = match[2].trim();

      if (this.releaseFilter) {
        this.releaseFilter.OnParseLine(key, value);
      } else if (this.ircNetwork !== undefined) {
        this.ircNetwork.OnParseLine(key, value);
      } else if (this.ircChannel !== undefined) {
        this.ircChannel.OnParseLine(key, value);
      }
    });

    // Save the remainder
    this.Save();

    // TODO: we don't support importing of irc networks/channels
    /*this.ircChannels.forEach((channel) => {
      let foundNetwork = false;
      for (let i = 0; i < this.ircNetworks.length; ++i) {
        if (channel.serverName === this.ircNetworks[i].values["server"]) {
          this.ircNetworks[i].values["channels"].push(channel.values);
        }
      }

      if (!foundNetwork) {
        this.warnings.push(`Failed to find related IRC network for channel '${channel.serverName}'`);
      }
    });*/
  }
}

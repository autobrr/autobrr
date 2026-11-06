/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

type MatrixAuthMethod = "" | "OAUTH" | "TOKEN";

type MatrixLoginState = "PENDING" | "SUCCEEDED" | "FAILED";

interface MatrixNetwork {
  id: number;
  name: string;
  enabled: boolean;
  homeserver: string;
  auth_method: MatrixAuthMethod;
  user_id: string;
  device_id: string;
  access_token: string;
  rooms: MatrixRoom[];
  health?: MatrixNetworkHealth;
}

type MatrixNetworkCreate = Omit<MatrixNetwork, "id" | "auth_method" | "user_id" | "device_id" | "health">;

interface MatrixRoom {
  id: number;
  enabled: boolean;
  name: string;
  room_id: string;
  health?: MatrixRoomHealth;
}

interface MatrixNetworkHealth {
  connected: boolean;
  connected_since?: string;
  healthy: boolean;
  error?: string;
  login_required?: boolean;
  login?: MatrixLogin;
}

interface MatrixRoomHealth {
  joined: boolean;
  joined_since?: string;
  last_announce?: string;
  error?: string;
}

interface MatrixLogin {
  state: MatrixLoginState;
  user_code?: string;
  verification_uri?: string;
  verification_uri_complete?: string;
  expires_at: string;
  user_id?: string;
  error?: string;
}

interface MatrixMessage {
  id: string;
  network: number;
  room: string;
  sender: string;
  msg: string;
  time: string;
}

interface MatrixHealthEvent {
  network: number;
  health: MatrixNetworkHealth;
  rooms: Record<string, MatrixRoomHealth>;
}

/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import { APIClient } from "@api/APIClient";
import { MatrixLoginForm, MatrixNetworkUpdateForm } from "@forms/settings/MatrixForms";
import "@app/i18n";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

const network: MatrixNetwork = {
  id: 5,
  name: "Watchtower",
  enabled: true,
  homeserver: "https://matrix.watchtower.pm",
  auth_method: "",
  user_id: "",
  device_id: "",
  access_token: "",
  rooms: [{ id: 1, enabled: true, name: "#blu-announce:matrix.watchtower.pm", room_id: "" }]
};

const pendingLogin: MatrixLogin = {
  state: "PENDING",
  user_code: "ABCD-EFGH",
  verification_uri: "https://auth.watchtower.pm/link",
  verification_uri_complete: "https://auth.watchtower.pm/link?code=ABCD-EFGH",
  expires_at: "2026-11-05T12:00:00Z"
};

const renderWithClient = (node: React.ReactNode) =>
  render(<QueryClientProvider client={new QueryClient()}>{node}</QueryClientProvider>);

test("device login shows the user code and the approval link, then the logged in user", async () => {
  const getByID = vi.spyOn(APIClient.matrix, "getByID").mockResolvedValue({ ...network, health: { connected: false, healthy: false, login: pendingLogin } });
  const login = vi.spyOn(APIClient.matrix, "login").mockResolvedValue(pendingLogin);

  renderWithClient(<MatrixLoginForm isOpen={true} toggle={() => {}} data={network} />);

  fireEvent.click(screen.getByText("Start login"));

  expect(await screen.findByText("ABCD-EFGH")).toBeTruthy();
  expect(login).toHaveBeenCalledWith(5);

  const link = screen.getByText("https://auth.watchtower.pm/link").closest("a") as HTMLAnchorElement;
  expect(link.href).toBe("https://auth.watchtower.pm/link?code=ABCD-EFGH");

  getByID.mockResolvedValue({
    ...network,
    user_id: "@autobrr:matrix.watchtower.pm",
    health: { connected: false, healthy: false, login: { state: "SUCCEEDED", expires_at: pendingLogin.expires_at, user_id: "@autobrr:matrix.watchtower.pm" } }
  });

  expect(await screen.findByText(/Logged in as @autobrr:matrix.watchtower.pm/, {}, { timeout: 5000 })).toBeTruthy();
});

test("a stale finished login is not shown before a new one starts", async () => {
  vi.spyOn(APIClient.matrix, "getByID").mockResolvedValue({
    ...network,
    health: { connected: false, healthy: false, login: { state: "FAILED", expires_at: pendingLogin.expires_at, error: "login code expired" } }
  });

  renderWithClient(<MatrixLoginForm isOpen={true} toggle={() => {}} data={network} />);

  expect(await screen.findByText("Start login")).toBeTruthy();
  expect(screen.queryByText(/login code expired/)).toBeNull();
});

test("a pasted access token is sent to the network", async () => {
  vi.spyOn(APIClient.matrix, "getByID").mockResolvedValue(network);
  const setAccessToken = vi.spyOn(APIClient.matrix, "setAccessToken").mockResolvedValue();

  renderWithClient(<MatrixLoginForm isOpen={true} toggle={() => {}} data={network} />);

  fireEvent.change(document.querySelector("input[name='access_token']") as HTMLInputElement, { target: { value: "syt_token" } });
  fireEvent.click(screen.getByText("Save token"));

  await waitFor(() => expect(setAccessToken).toHaveBeenCalledWith(5, "syt_token"));
});

test("update form rejects a room that is not an alias or room id", async () => {
  const update = vi.spyOn(APIClient.matrix, "update").mockResolvedValue();

  renderWithClient(<MatrixNetworkUpdateForm isOpen={true} toggle={() => {}} data={network} onLogin={() => {}} />);

  const room = document.querySelector("input[name='rooms[0].name']") as HTMLInputElement;
  fireEvent.change(room, { target: { value: "blu-announce" } });
  fireEvent.click(screen.getByText("Save"));

  await waitFor(() => expect(room.className).toContain("border-red-500"));
  expect(update).not.toHaveBeenCalled();
});

test("a pending login started earlier is shown when the panel opens again", async () => {
  vi.spyOn(APIClient.matrix, "getByID").mockResolvedValue({ ...network, health: { connected: false, healthy: false, login: pendingLogin } });
  const login = vi.spyOn(APIClient.matrix, "login");

  renderWithClient(<MatrixLoginForm isOpen={true} toggle={() => {}} data={network} />);

  expect(await screen.findByText("ABCD-EFGH")).toBeTruthy();
  expect(screen.queryByText("Start login")).toBeNull();
  expect(login).not.toHaveBeenCalled();
});

test("the edit panel account section logs in and out without saving the form", async () => {
  const update = vi.spyOn(APIClient.matrix, "update").mockResolvedValue();
  const logout = vi.spyOn(APIClient.matrix, "logout").mockResolvedValue();
  const onLogin = vi.fn();

  renderWithClient(
    <MatrixNetworkUpdateForm
      isOpen={true}
      toggle={() => {}}
      data={{ ...network, auth_method: "OAUTH", user_id: "@autobrr:matrix.watchtower.pm" }}
      onLogin={onLogin}
    />
  );

  expect(await screen.findByText("@autobrr:matrix.watchtower.pm")).toBeTruthy();
  expect(screen.getByText("Browser login")).toBeTruthy();

  fireEvent.click(screen.getByRole("button", { name: "Log in" }));
  expect(onLogin).toHaveBeenCalledTimes(1);

  fireEvent.click(screen.getByRole("button", { name: "Log out" }));
  await waitFor(() => expect(logout).toHaveBeenCalledWith(5));

  expect(update).not.toHaveBeenCalled();
});

test("the edit panel offers no log out without a login", async () => {
  renderWithClient(<MatrixNetworkUpdateForm isOpen={true} toggle={() => {}} data={network} onLogin={() => {}} />);

  expect(await screen.findByText("Not logged in")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Log out" })).toBeNull();
});

/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Suspense } from "react";
import { BrowserRouter, Outlet, Route, Routes } from "react-router-dom";

import { baseUrl } from "@utils";

import { Header } from "@components/header";
import { SectionLoader } from "@components/SectionLoader";
import { NotFound } from "@components/alerts/NotFound";

import { Logs } from "@screens/Logs";
import { Releases } from "@screens/Releases";
import { Settings } from "@screens/Settings";
import { Dashboard } from "@screens/Dashboard";
import { Login, Onboarding } from "@screens/auth";
import { Filters, FilterDetails } from "@screens/filters";
import * as SettingsSubPage from "@screens/settings/index";

const BaseLayout = () => (
  <div className="min-h-screen">
    <Header />
    <Suspense fallback={<SectionLoader $size="xlarge" />}>
      <Outlet />
    </Suspense>
  </div>
);

export const LocalRouter = ({ isLoggedIn }: { isLoggedIn: boolean }) => (
  <BrowserRouter basename={baseUrl()}>
    {isLoggedIn ? (
      <Routes>
        <Route path="*" element={<NotFound />} />
        <Route element={<BaseLayout />}>
          <Route index element={<Dashboard />} />
          <Route path="logs" element={<Logs />} />
          <Route path="releases" element={<Releases />} />
          <Route path="filters">
            <Route index element={<Filters />} />
            <Route path=":filterId/*" element={<FilterDetails />} />
          </Route>
          <Route path="settings" element={<Settings />}>
            <Route index element={<SettingsSubPage.Application />} />
            <Route path="logs" element={<SettingsSubPage.Logs />} />
            <Route path="api-keys" element={<SettingsSubPage.Api />} />
            <Route path="indexers" element={<SettingsSubPage.Indexer />} />
            <Route path="feeds" element={<SettingsSubPage.Feed />} />
            <Route path="irc" element={<SettingsSubPage.Irc />} />
            <Route path="clients" element={<SettingsSubPage.DownloadClient />} />
            <Route path="notifications" element={<SettingsSubPage.Notification />} />
            <Route path="releases" element={<SettingsSubPage.Release />} />
            <Route path="omegabrr" element={<SettingsSubPage.Omegabrr />} />
            <Route path="regex-playground" element={<SettingsSubPage.RegexPlayground />} />
            <Route path="account" element={<SettingsSubPage.Account />} />
          </Route>
        </Route>
      </Routes>
    ) : (
      <Routes>
        <Route path="/onboard" element={<Onboarding />} />
        <Route path="*" element={<Login />} />
      </Routes>
    )}
  </BrowserRouter>
);

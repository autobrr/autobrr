/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Suspense } from "react";
import { useTranslation } from "react-i18next";

import { ChartCard, ChartSkeleton } from "./charts";
import type { DashboardWidgetDef } from "./widgets";

const WidgetFallback = ({ def }: { def: DashboardWidgetDef }) => {
  const { t } = useTranslation("common");

  return (
    <ChartCard title={t(def.titleKey)}>
      <ChartSkeleton heightClass="h-56" />
    </ChartCard>
  );
};

export const WidgetBoundary = ({ def }: { def: DashboardWidgetDef }) => (
  <Suspense fallback={<WidgetFallback def={def} />}>
    <def.Component />
  </Suspense>
);

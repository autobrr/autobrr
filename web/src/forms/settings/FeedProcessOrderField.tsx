/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMemo } from "react";
import { useTranslation } from "react-i18next";

import { SelectFieldBasic } from "@components/inputs/select_wide";
import { OptionBasicTyped } from "@domain/constants";

interface FeedProcessOrderFieldProps {
  name: string;
}

export function FeedProcessOrderField({ name }: FeedProcessOrderFieldProps) {
  const { t } = useTranslation("settings");
  const options = useMemo<OptionBasicTyped<FeedProcessOrder>[]>(() => [
    { label: t("forms.feed.processOrderOldestFirst"), value: "OLDEST_FIRST" },
    { label: t("forms.feed.processOrderNewestFirst"), value: "NEWEST_FIRST" }
  ], [t]);

  return (
    <SelectFieldBasic name={name} label={t("forms.feed.processOrder")} help={t("forms.feed.processOrderHelp")} options={options} />
  );
}

/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { ActionRtorrentRenameOptions } from "@domain/constants";
import { FilterHalfRow, FilterLayout, FilterSection } from "@screens/filters/sections/_components.tsx";
import { DownloadClientSelect, Select, SwitchGroup, TextAreaAutoResize, TextField } from "@components/inputs";
import { useTranslation } from "react-i18next";


export const RTorrent = ({ idx, action, clients }: ClientActionProps) => {
  const { t } = useTranslation("filters");

  return (
  <>
    <FilterSection
      title={t("actionComponents.instance.title")}
      subtitle={t("actionComponents.instance.subtitle")}
    >
      <FilterLayout>
        <FilterHalfRow>
          <DownloadClientSelect
            name={`actions.${idx}.client_id`}
            action={action}
            clients={clients}
          />
        </FilterHalfRow>

        <FilterHalfRow>
          <TextField
            name={`actions.${idx}.label`}
            label={t("actionComponents.rtorrent.label")}
            columns={6}
            placeholder={t("actionComponents.rtorrent.labelPlaceholder")}
          />
        </FilterHalfRow>
      </FilterLayout>

      <TextAreaAutoResize
        name={`actions.${idx}.save_path`}
        label={t("actionComponents.common.savePath")}
        placeholder={t("actionComponents.common.savePathPlaceholder")}
      />

      <FilterLayout>
        <FilterHalfRow>
          <SwitchGroup
            name={`actions.${idx}.paused`}
            label={t("actionComponents.common.addPaused")}
            description={t("actionComponents.common.addPausedDescription")}
            className="pt-2 pb-4"
          />
          <Select
            name={`actions.${idx}.content_layout`}
            label={t("actionComponents.rtorrent.renameToPath")}
            optionDefaultText={t("actionComponents.rtorrent.renameDefault")}
            options={ActionRtorrentRenameOptions}
          />
        </FilterHalfRow>
      </FilterLayout>
    </FilterSection>
  </>
  );
};

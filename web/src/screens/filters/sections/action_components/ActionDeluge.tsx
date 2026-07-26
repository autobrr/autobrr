/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { CollapsibleSection, FilterHalfRow, FilterLayout, FilterSection } from "../_components";
import { DownloadClientSelect, NumberField, SwitchGroup, TextAreaAutoResize, TextField } from "@components/inputs";
import { useTranslation } from "react-i18next";

export const Deluge = ({ idx, action, clients }: ClientActionProps) => {
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
            label={t("actionComponents.deluge.label")}
            columns={6}
            placeholder={t("actionComponents.deluge.labelPlaceholder")}
          />
        </FilterHalfRow>

        <TextAreaAutoResize
          name={`actions.${idx}.save_path`}
          label={t("actionComponents.common.savePath")}
          placeholder={t("actionComponents.common.savePathPlaceholder")}
        />
      </FilterLayout>

      <FilterLayout className="pb-6">
        <FilterHalfRow>
          <SwitchGroup
            name={`actions.${idx}.paused`}
            label={t("actionComponents.common.addPaused")}
            description={t("actionComponents.common.addPausedDescription")}
          />
        </FilterHalfRow>
        <FilterHalfRow>
        <SwitchGroup
            name={`actions.${idx}.skip_hash_check`}
            label={t("actionComponents.common.skipHashCheck")}
            description={t("actionComponents.common.skipHashCheckDescription")}
            tooltip={<div>{t("actionComponents.deluge.skipHashCheckTooltip")}</div>}
          />
        </FilterHalfRow>
      </FilterLayout>

      <CollapsibleSection
        noBottomBorder
        title={t("actionComponents.common.limitsTitle")}
        subtitle={t("actionComponents.common.limitsSubtitle")}
      >
        <NumberField
          name={`actions.${idx}.limit_download_speed`}
          label={t("actionComponents.common.limitDownloadKb")}
          placeholder={t("actionComponents.common.numberNoLimit")}
        />
        <NumberField
          name={`actions.${idx}.limit_upload_speed`}
          label={t("actionComponents.common.limitUploadKb")}
          placeholder={t("actionComponents.common.numberNoLimit")}
        />
      </CollapsibleSection>
    </FilterSection>
  </>
  );
};

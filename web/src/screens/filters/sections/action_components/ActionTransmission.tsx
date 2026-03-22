/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { CollapsibleSection, FilterHalfRow, FilterLayout, FilterSection, FilterWideGridGapClass } from "../_components";
import { DownloadClientSelect, NumberField, SwitchGroup, TextAreaAutoResize, TextField } from "@components/inputs";
import { useTranslation } from "react-i18next";

export const Transmission = ({ idx, action, clients }: ClientActionProps) => {
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
            label={t("actionComponents.transmission.label")}
            columns={6}
            placeholder={t("actionComponents.transmission.labelPlaceholder")}
          />
        </FilterHalfRow>
      </FilterLayout>

      <TextAreaAutoResize
        name={`actions.${idx}.save_path`}
        label={t("actionComponents.common.savePath")}
        columns={6}
        placeholder={t("actionComponents.common.savePathPlaceholder")}
      />

      <FilterLayout className="pb-6">
        <FilterHalfRow>
          <SwitchGroup
            name={`actions.${idx}.paused`}
            label={t("actionComponents.common.addPaused")}
            description={t("actionComponents.common.addPausedDescription")}
          />
        </FilterHalfRow>
      </FilterLayout>

      <CollapsibleSection
        title={t("actionComponents.common.limitsTitle")}
        subtitle={t("actionComponents.common.limitsSubtitle")}
      >
        <FilterLayout>
          <NumberField
            name={`actions.${idx}.limit_download_speed`}
            label={t("actionComponents.common.limitDownloadKib")}
            placeholder={t("actionComponents.common.numberNoLimit")}
          />
          <NumberField
            name={`actions.${idx}.limit_upload_speed`}
            label={t("actionComponents.common.limitUploadKib")}
            placeholder={t("actionComponents.common.numberNoLimit")}
          />
        </FilterLayout>

        <FilterLayout>
          <NumberField
            name={`actions.${idx}.limit_ratio`}
            label={t("actionComponents.common.ratioLimit")}
            placeholder={t("actionComponents.common.numberNoLimit")}
            step={0.25}
            isDecimal
          />
          <NumberField
            name={`actions.${idx}.limit_seed_time`}
            label={t("actionComponents.common.seedTimeMinutes")}
            placeholder={t("actionComponents.common.numberNoLimit")}
          />
        </FilterLayout>
      </CollapsibleSection>

      <CollapsibleSection
        noBottomBorder
        title={t("actionComponents.common.announceTitle")}
        subtitle={t("actionComponents.common.announceSubtitle")}
        childClassName={FilterWideGridGapClass}
      >
        <FilterHalfRow>
          <SwitchGroup
            name={`actions.${idx}.reannounce_skip`}
            label={t("actionComponents.common.disableReannounce")}
            description={t("actionComponents.common.disableReannounceDescription")}
            className="pt-2 pb-4"
          />
          <NumberField
            name={`actions.${idx}.reannounce_interval`}
            label={t("actionComponents.common.reannounceInterval")}
            placeholder={t("actionComponents.common.reannounceIntervalPlaceholder")}
          />
        </FilterHalfRow>
        <FilterHalfRow>
          <SwitchGroup
            name={`actions.${idx}.reannounce_delete`}
            label={t("actionComponents.common.deleteStalled")}
            description={t("actionComponents.common.deleteStalledDescription")}
            className="pt-2 pb-4"
          />
          <NumberField
            name={`actions.${idx}.reannounce_max_attempts`}
            label={t("actionComponents.common.reannounceMaxAttempts")}
          />
        </FilterHalfRow>
      </CollapsibleSection>
    </FilterSection>
  </>
  );
};

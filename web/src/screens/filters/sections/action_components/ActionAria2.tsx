/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { CollapsibleSection, FilterHalfRow, FilterLayout, FilterSection } from "../_components";
import { DownloaderSelect, NumberField, SwitchGroup, TextAreaAutoResize } from "@components/inputs";
import { useTranslation } from "react-i18next";

export const Aria2 = ({ idx, action, clients }: ClientActionProps) => {
  const { t } = useTranslation("filters");

  return (
  <>
    <FilterSection
      title={t("actionComponents.instance.title")}
      subtitle={t("actionComponents.instance.subtitle")}
    >
      <FilterLayout>
        <FilterHalfRow>
          <DownloaderSelect
            name={`actions.${idx}.client_id`}
            action={action}
            clients={clients}
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
      </FilterLayout>

      <CollapsibleSection
        noBottomBorder
        title={t("actionComponents.common.limitsTitle")}
        subtitle={t("actionComponents.common.limitsSubtitle")}
      >
        <FilterHalfRow>
          <NumberField
            name={`actions.${idx}.limit_download_speed`}
            label={t("actionComponents.common.limitDownloadKib")}
            placeholder={t("actionComponents.common.numberNoLimit")}
          />
        </FilterHalfRow>
        <FilterHalfRow>
          <NumberField
            name={`actions.${idx}.limit_upload_speed`}
            label={t("actionComponents.common.limitUploadKib")}
            placeholder={t("actionComponents.common.numberNoLimit")}
          />
        </FilterHalfRow>
        <FilterHalfRow>
          <NumberField
            name={`actions.${idx}.limit_ratio`}
            label={t("actionComponents.common.ratioLimit")}
            placeholder={t("actionComponents.common.numberNoLimit")}
            step={0.25}
            isDecimal
          />
        </FilterHalfRow>
        <FilterHalfRow>
          <NumberField
            name={`actions.${idx}.limit_seed_time`}
            label={t("actionComponents.common.seedTimeMinutes")}
            placeholder={t("actionComponents.common.numberNoLimit")}
          />
        </FilterHalfRow>
      </CollapsibleSection>
    </FilterSection>
  </>
  );
};

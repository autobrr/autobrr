/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { CollapsibleSection, FilterHalfRow, FilterLayout, FilterSection } from "../_components";
import { DownloadClientSelect, NumberField, TextAreaAutoResize, TextField } from "@components/inputs";
import { useTranslation } from "react-i18next";

export const Porla = ({ idx, action, clients }: ClientActionProps) => {
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
            label={t("actionComponents.porla.preset")}
            placeholder={t("actionComponents.porla.presetPlaceholder")}
            tooltip={
              <div>{t("actionComponents.porla.presetTooltip")}</div>
            }
          />
        </FilterHalfRow>
      </FilterLayout>

      <TextAreaAutoResize
        name={`actions.${idx}.save_path`}
        label={t("actionComponents.common.savePath")}
        placeholder={t("actionComponents.porla.savePathPlaceholder")}
        className="pb-6"
      />

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
      </CollapsibleSection>
    </FilterSection>
  </>
  );
};

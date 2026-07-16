/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { DocsLink } from "@components/ExternalLink";
import { ActionContentLayoutOptions, ActionPriorityOptions } from "@domain/constants";

import { CollapsibleSection, FilterHalfRow, FilterLayout, FilterSection, FilterWideGridGapClass } from "../_components";
import {
  DownloadClientSelect,
  NumberField,
  Select,
  SwitchGroup,
  TextAreaAutoResize,
  TextField
} from "@components/inputs";

export const QBittorrent = ({ idx, action, clients }: ClientActionProps) => {
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
      </FilterLayout>

      <FilterLayout>
        <TextField
          name={`actions.${idx}.category`}
          label={t("actionComponents.common.category")}
          columns={6}
          placeholder={t("actionComponents.common.categoryPlaceholder")}
          tooltip={
            <div>
              <p>{t("actionComponents.qbittorrent.macrosTooltip")}</p>
              <DocsLink href="https://autobrr.com/filters/macros" />
            </div>
          }
        />

        <TextField
          name={`actions.${idx}.tags`}
          label={t("actionComponents.qbittorrent.tags")}
          columns={6}
          placeholder={t("actionComponents.qbittorrent.tagsPlaceholder")}
          tooltip={
            <div>
              <p>{t("actionComponents.qbittorrent.macrosTooltip")}</p>
              <DocsLink href="https://autobrr.com/filters/macros" />
            </div>
          }
        />
      </FilterLayout>

      <FilterLayout className="pb-6">
        <FilterHalfRow>
          <TextAreaAutoResize
            name={`actions.${idx}.save_path`}
            label={t("actionComponents.common.savePath")}
            placeholder={t("actionComponents.qbittorrent.savePathPlaceholder")}
            tooltip={
              <div>
                <p>{t("actionComponents.qbittorrent.savePathTooltip")}</p>
                <br />
                <p>{t("actionComponents.qbittorrent.macrosTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters/macros" />
              </div>
            }
          />
        </FilterHalfRow>
        <FilterHalfRow>
          <TextAreaAutoResize
            name={`actions.${idx}.download_path`}
            label={t("actionComponents.qbittorrent.downloadPath")}
            placeholder={t("actionComponents.qbittorrent.downloadPathPlaceholder")}
            tooltip={
              <div>
                <p>{t("actionComponents.qbittorrent.downloadPathTooltip")}</p>
                <br />
                <p>{t("actionComponents.qbittorrent.macrosTooltip")}</p>
                <DocsLink href="https://autobrr.com/filters/macros" />
              </div>
            }
          />
        </FilterHalfRow>
      </FilterLayout>

      <CollapsibleSection
        title={t("actionComponents.qbittorrent.rulesTitle")}
        subtitle={t("actionComponents.qbittorrent.rulesSubtitle")}
        childClassName={FilterWideGridGapClass}
      >
        <FilterHalfRow>
          <SwitchGroup
            name={`actions.${idx}.ignore_rules`}
            label={t("actionComponents.qbittorrent.ignoreRules")}
            description={
              <p>
                {t("actionComponents.qbittorrent.ignoreRulesDescription")} <Link className="text-blue-400 visited:text-blue-400" to="/settings/clients">Client Settings</Link>.
              </p>
            }
            className="py-2 pb-4"
          />
          <Select
            name={`actions.${idx}.content_layout`}
            label={t("actionComponents.qbittorrent.contentLayout")}
            optionDefaultText={t("actionComponents.qbittorrent.contentLayoutDefault")}
            options={ActionContentLayoutOptions}
            className="py-2 pb-4"
          />
          <Select
            name={`actions.${idx}.priority`}
            label={t("actionComponents.qbittorrent.priority")}
            optionDefaultText={t("actionComponents.qbittorrent.priorityDefault")}
            options={ActionPriorityOptions}
            tooltip={
              <div>
                <p>{t("actionComponents.qbittorrent.priorityTooltip")}</p>
              </div>
            }
          />
        </FilterHalfRow>

        <FilterHalfRow>
          <SwitchGroup
            name={`actions.${idx}.paused`}
            label={t("actionComponents.common.addPaused")}
            description={t("actionComponents.common.addPausedDescription")}
          />
          <SwitchGroup
            name={`actions.${idx}.skip_hash_check`}
            label={t("actionComponents.common.skipHashCheck")}
            description={t("actionComponents.common.skipHashCheckDescription")}
            className="pt-4 sm:pt-4"
          />
          <SwitchGroup
            name={`actions.${idx}.first_last_piece_prio`}
            label={t("actionComponents.qbittorrent.firstLastPiece")}
            description={t("actionComponents.qbittorrent.firstLastPieceDescription")}
            className="pt-6 sm:pt-10"
          />
        </FilterHalfRow>
      </CollapsibleSection>

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

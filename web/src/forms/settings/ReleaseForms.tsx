/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { APIClient } from "@api/APIClient.ts";
import { ReleaseProfileDuplicateKeys } from "@api/query_keys.ts";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast.tsx";
import { SwitchGroupWide, TextFieldWide } from "@components/inputs";
import { SlideOver } from "@components/panels";
import { AddFormProps, UpdateFormProps } from "@forms/_shared";

export function ReleaseProfileDuplicateAddForm({ isOpen, toggle }: AddFormProps) {
  const { t } = useTranslation("settings");
  const queryClient = useQueryClient();

  const addMutation = useMutation({
    mutationFn: (profile: ReleaseProfileDuplicate) => APIClient.release.profiles.duplicates.store(profile),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ReleaseProfileDuplicateKeys.lists() });
      toast.custom((toastInstance) => <Toast type="success" body={t("forms.releaseProfile.added")} t={toastInstance} />);

      toggle();
    },
    onError: () => {
      toast.custom((toastInstance) => <Toast type="error" body={t("forms.releaseProfile.addFailed")} t={toastInstance} />);
    }
  });

  const onSubmit = (data: unknown) => addMutation.mutate(data as ReleaseProfileDuplicate);

  const initialValues: ReleaseProfileDuplicate = {
    id: 0,
    name: "",
    protocol: false,
    release_name: false,
    hash: false,
    title: false,
    sub_title: false,
    year: false,
    month: false,
    day: false,
    source: false,
    resolution: false,
    codec: false,
    container: false,
    dynamic_range: false,
    audio: false,
    group: false,
    season: false,
    episode: false,
    website: false,
    proper: false,
    repack: false,
    edition: false,
    hybrid: false,
    language: false,
  };

  return (
    <SlideOver
      type="CREATE"
      title={t("forms.releaseProfile.title")}
      isOpen={isOpen}
      toggle={toggle}
      onSubmit={onSubmit}
      initialValues={initialValues}
    >
      {() => (
        <div className="py-2 space-y-6 sm:py-0 sm:space-y-0 divide-y divide-gray-200 dark:divide-gray-700">
          <TextFieldWide required name="name" label={t("forms.releaseProfile.fields.name")}/>

          <SwitchGroupWide name="release_name" label={t("forms.releaseProfile.fields.releaseName")} description={t("forms.releaseProfile.descriptions.releaseName")} />
          <SwitchGroupWide name="hash" label={t("forms.releaseProfile.fields.hash")} description={t("forms.releaseProfile.descriptions.hash")} />
          <SwitchGroupWide name="title" label={t("forms.releaseProfile.fields.title")} description={t("forms.releaseProfile.descriptions.title")} />
          <SwitchGroupWide name="sub_title" label={t("forms.releaseProfile.fields.subTitle")} description={t("forms.releaseProfile.descriptions.subTitle")} />
          <SwitchGroupWide name="year" label={t("forms.releaseProfile.fields.year")} />
          <SwitchGroupWide name="month" label={t("forms.releaseProfile.fields.month")} description={t("forms.releaseProfile.descriptions.dailyReleases")} />
          <SwitchGroupWide name="day" label={t("forms.releaseProfile.fields.day")} description={t("forms.releaseProfile.descriptions.dailyReleases")} />
          <SwitchGroupWide name="source" label={t("forms.releaseProfile.fields.source")} />
          <SwitchGroupWide name="resolution" label={t("forms.releaseProfile.fields.resolution")} />
          <SwitchGroupWide name="codec" label={t("forms.releaseProfile.fields.codec")} />
          <SwitchGroupWide name="container" label={t("forms.releaseProfile.fields.container")} />
          <SwitchGroupWide name="dynamic_range" label={t("forms.releaseProfile.fields.dynamicRange")} />
          <SwitchGroupWide name="audio" label={t("forms.releaseProfile.fields.audio")} />
          <SwitchGroupWide name="group" label={t("forms.releaseProfile.fields.group")} description={t("forms.releaseProfile.descriptions.group")} />
          <SwitchGroupWide name="season" label={t("forms.releaseProfile.fields.season")} />
          <SwitchGroupWide name="episode" label={t("forms.releaseProfile.fields.episode")} />
          <SwitchGroupWide name="website" label={t("forms.releaseProfile.fields.website")} description={t("forms.releaseProfile.descriptions.website")} />
          <SwitchGroupWide name="proper" label={t("forms.releaseProfile.fields.proper")} />
          <SwitchGroupWide name="repack" label={t("forms.releaseProfile.fields.repack")} />
          <SwitchGroupWide name="edition" label={t("forms.releaseProfile.fields.edition")} />
          <SwitchGroupWide name="hybrid" label={t("forms.releaseProfile.fields.hybrid")} />
          <SwitchGroupWide name="language" label={t("forms.releaseProfile.fields.language")} />
        </div>
      )}
    </SlideOver>
  );
}

export function ReleaseProfileDuplicateUpdateForm({ isOpen, toggle, data: profile }: UpdateFormProps<ReleaseProfileDuplicate>) {
  const { t } = useTranslation("settings");
  const queryClient = useQueryClient();

  const storeMutation = useMutation({
    mutationFn: (profile: ReleaseProfileDuplicate) => APIClient.release.profiles.duplicates.store(profile),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ReleaseProfileDuplicateKeys.lists() });
      toast.custom((toastInstance) => <Toast type="success" body={t("forms.releaseProfile.added")} t={toastInstance} />);

      toggle();
    },
    onError: () => {
      toast.custom((toastInstance) => <Toast type="error" body={t("forms.releaseProfile.addFailed")} t={toastInstance} />);
    }
  });

  const onSubmit = (data: unknown) => storeMutation.mutate(data as ReleaseProfileDuplicate);

  const deleteMutation = useMutation({
    mutationFn: (profileId: number) => APIClient.release.profiles.duplicates.delete(profileId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ReleaseProfileDuplicateKeys.lists() });
      queryClient.invalidateQueries({ queryKey: ReleaseProfileDuplicateKeys.detail(profile.id) });

      toast.custom((toastInstance) => <Toast type="success" body={t("forms.releaseProfile.deleted", { name: profile.name })} t={toastInstance} />);

      toggle();
    },
  });

  const onDelete = () => deleteMutation.mutate(profile.id);

  const initialValues: ReleaseProfileDuplicate = {
    id: profile.id,
    name: profile.name,
    protocol: profile.protocol,
    release_name: profile.release_name,
    hash: profile.hash,
    title: profile.title,
    sub_title: profile.sub_title,
    year: profile.year,
    month: profile.month,
    day: profile.day,
    source: profile.source,
    resolution: profile.resolution,
    codec: profile.codec,
    container: profile.container,
    dynamic_range: profile.dynamic_range,
    audio: profile.audio,
    group: profile.group,
    season: profile.season,
    episode: profile.episode,
    website: profile.website,
    proper: profile.proper,
    repack: profile.repack,
    edition: profile.edition,
    hybrid: profile.hybrid,
    language: profile.language,
  };

  return (
    <SlideOver
      type="UPDATE"
      title={t("forms.releaseProfile.title")}
      isOpen={isOpen}
      toggle={toggle}
      deleteAction={onDelete}
      onSubmit={onSubmit}
      initialValues={initialValues}
    >
      {() => (
        <div className="py-2 space-y-6 sm:py-0 sm:space-y-0 divide-y divide-gray-200 dark:divide-gray-700">
          <TextFieldWide required name="name" label={t("forms.releaseProfile.fields.name")}/>

          <SwitchGroupWide name="release_name" label={t("forms.releaseProfile.fields.releaseName")} description={t("forms.releaseProfile.descriptions.releaseName")} />
          <SwitchGroupWide name="hash" label={t("forms.releaseProfile.fields.hash")} description={t("forms.releaseProfile.descriptions.hash")} />
          <SwitchGroupWide name="title" label={t("forms.releaseProfile.fields.title")} description={t("forms.releaseProfile.descriptions.title")} />
          <SwitchGroupWide name="sub_title" label={t("forms.releaseProfile.fields.subTitle")} description={t("forms.releaseProfile.descriptions.subTitle")} />
          <SwitchGroupWide name="year" label={t("forms.releaseProfile.fields.year")} />
          <SwitchGroupWide name="month" label={t("forms.releaseProfile.fields.month")} description={t("forms.releaseProfile.descriptions.dailyReleases")} />
          <SwitchGroupWide name="day" label={t("forms.releaseProfile.fields.day")} description={t("forms.releaseProfile.descriptions.dailyReleases")} />
          <SwitchGroupWide name="source" label={t("forms.releaseProfile.fields.source")} />
          <SwitchGroupWide name="resolution" label={t("forms.releaseProfile.fields.resolution")} />
          <SwitchGroupWide name="codec" label={t("forms.releaseProfile.fields.codec")} />
          <SwitchGroupWide name="container" label={t("forms.releaseProfile.fields.container")} />
          <SwitchGroupWide name="dynamic_range" label={t("forms.releaseProfile.fields.dynamicRange")} description={t("forms.releaseProfile.descriptions.dynamicRange")} />
          <SwitchGroupWide name="audio" label={t("forms.releaseProfile.fields.audio")} />
          <SwitchGroupWide name="group" label={t("forms.releaseProfile.fields.group")} description={t("forms.releaseProfile.descriptions.group")} />
          <SwitchGroupWide name="season" label={t("forms.releaseProfile.fields.season")} />
          <SwitchGroupWide name="episode" label={t("forms.releaseProfile.fields.episode")} />
          <SwitchGroupWide name="website" label={t("forms.releaseProfile.fields.website")} description={t("forms.releaseProfile.descriptions.website")} />
          <SwitchGroupWide name="repack" label={t("forms.releaseProfile.fields.repack")} />
          <SwitchGroupWide name="proper" label={t("forms.releaseProfile.fields.proper")} />
          <SwitchGroupWide name="edition" label={t("forms.releaseProfile.fields.edition")} description={t("forms.releaseProfile.descriptions.edition")} />
          <SwitchGroupWide name="hybrid" label={t("forms.releaseProfile.fields.hybrid")} />
          <SwitchGroupWide name="language" label={t("forms.releaseProfile.fields.language")} description={t("forms.releaseProfile.descriptions.language")} />
        </div>
      )}
    </SlideOver>
  );
}

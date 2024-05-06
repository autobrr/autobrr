/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { APIClient } from "@api/APIClient.ts";
import { ReleaseProfileDuplicateKeys } from "@api/query_keys.ts";
import { toast } from "react-hot-toast";
import Toast from "@components/notifications/Toast.tsx";
import { SwitchGroupWide, TextFieldWide } from "@components/inputs";
import { SlideOver } from "@components/panels";
import { AddFormProps, UpdateFormProps } from "@forms/_shared";

export function ReleaseProfileDuplicateAddForm({ isOpen, toggle }: AddFormProps) {
  const queryClient = useQueryClient();

  const addMutation = useMutation({
    mutationFn: (profile: ReleaseProfileDuplicate) => APIClient.release.profiles.duplicates.store(profile),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ReleaseProfileDuplicateKeys.lists() });
      toast.custom((t) => <Toast type="success" body="Profile was added" t={t} />);

      toggle();
    },
    onError: () => {
      toast.custom((t) => <Toast type="error" body="Profile could not be added" t={t} />);
    }
  });

  const onSubmit = (data: unknown) => addMutation.mutate(data as ReleaseProfileDuplicate);

  const initialValues: ReleaseProfileDuplicate = {
    id: 0,
    name: "",
    protocol: false,
    release_name: false,
    title: false,
    year: false,
    month: false,
    day: false,
    source: false,
    resolution: false,
    codec: false,
    container: false,
    hdr: false,
    group: false,
    season: false,
    episode: false,
    proper: false,
    repack: false
  };

  return (
    <SlideOver
      type="CREATE"
      title="Duplicate Profile"
      isOpen={isOpen}
      toggle={toggle}
      onSubmit={onSubmit}
      initialValues={initialValues}
    >
      {() => (
        <div className="py-2 space-y-6 sm:py-0 sm:space-y-0 divide-y divide-gray-200 dark:divide-gray-700">
          <TextFieldWide required name="name" label="Name"/>

          <SwitchGroupWide name="release_name" label="Release name" />
          <SwitchGroupWide name="title" label="Title" />
          <SwitchGroupWide name="year" label="Year" />
          <SwitchGroupWide name="month" label="Month" />
          <SwitchGroupWide name="day" label="Day" />
          <SwitchGroupWide name="source" label="Source" />
          <SwitchGroupWide name="resolution" label="Resolution" />
          <SwitchGroupWide name="codec" label="Codec" />
          <SwitchGroupWide name="container" label="Container" />
          <SwitchGroupWide name="hdr" label="HDR" />
          <SwitchGroupWide name="group" label="Group" />
          <SwitchGroupWide name="season" label="Season" />
          <SwitchGroupWide name="episode" label="Episode" />
        </div>
      )}
    </SlideOver>
  );
}

export function ReleaseProfileDuplicateUpdateForm({ isOpen, toggle, data: profile }: UpdateFormProps<ReleaseProfileDuplicate>) {
  const queryClient = useQueryClient();

  const storeMutation = useMutation({
    mutationFn: (profile: ReleaseProfileDuplicate) => APIClient.release.profiles.duplicates.store(profile),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ReleaseProfileDuplicateKeys.lists() });
      toast.custom((t) => <Toast type="success" body="Profile was added" t={t} />);

      toggle();
    },
    onError: () => {
      toast.custom((t) => <Toast type="error" body="Profile could not be added" t={t} />);
    }
  });

  const onSubmit = (data: unknown) => storeMutation.mutate(data as ReleaseProfileDuplicate);

  const deleteMutation = useMutation({
    mutationFn: (profileId: number) => APIClient.release.profiles.duplicates.delete(profileId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ReleaseProfileDuplicateKeys.lists() });
      queryClient.invalidateQueries({ queryKey: ReleaseProfileDuplicateKeys.detail(profile.id) });

      toast.custom((t) => <Toast type="success" body={`Profile ${profile.name} was deleted!`} t={t} />);

      toggle();
    },
  });

  const onDelete = () => deleteMutation.mutate(profile.id);

  const initialValues: ReleaseProfileDuplicate = {
    id: profile.id,
    name: profile.name,
    protocol: profile.protocol,
    release_name: profile.release_name,
    title: profile.title,
    year: profile.year,
    month: profile.month,
    day: profile.day,
    source: profile.source,
    resolution: profile.resolution,
    codec: profile.codec,
    container: profile.container,
    hdr: profile.hdr,
    group: profile.group,
    season: profile.season,
    episode: profile.episode,
    proper: profile.proper,
    repack: profile.repack
  };

  return (
    <SlideOver
      type="UPDATE"
      title="Duplicate Profile"
      isOpen={isOpen}
      toggle={toggle}
      deleteAction={onDelete}
      onSubmit={onSubmit}
      initialValues={initialValues}
    >
      {() => (
        <div className="py-2 space-y-6 sm:py-0 sm:space-y-0 divide-y divide-gray-200 dark:divide-gray-700">
          <TextFieldWide required name="name" label="Name"/>

          <SwitchGroupWide name="release_name" label="Release name" />
          <SwitchGroupWide name="title" label="Title" />
          <SwitchGroupWide name="year" label="Year" />
          <SwitchGroupWide name="month" label="Month" />
          <SwitchGroupWide name="day" label="Day" />
          <SwitchGroupWide name="source" label="Source" />
          <SwitchGroupWide name="resolution" label="Resolution" />
          <SwitchGroupWide name="codec" label="Codec" />
          <SwitchGroupWide name="container" label="Container" />
          <SwitchGroupWide name="hdr" label="HDR" />
          <SwitchGroupWide name="group" label="Group" />
          <SwitchGroupWide name="season" label="Season" />
          <SwitchGroupWide name="episode" label="Episode" />
        </div>
      )}
    </SlideOver>
  );
}

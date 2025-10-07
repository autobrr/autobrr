/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { APIClient } from "@api/APIClient.ts";
import { ReleaseProfileDuplicateKeys } from "@api/query_keys.ts";
import { toast } from "@components/hot-toast";
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
      title="Duplicate Profile"
      isOpen={isOpen}
      toggle={toggle}
      onSubmit={onSubmit}
      initialValues={initialValues}
    >
      {() => (
        <div className="py-2 space-y-6 sm:py-0 sm:space-y-0 divide-y divide-gray-200 dark:divide-gray-700">
          <TextFieldWide required name="name" label="Name"/>

          <SwitchGroupWide name="release_name" label="Release name" description="Full release name" />
          <SwitchGroupWide name="hash" label="Hash" description="Normalized hash of the release name. Use with Release name for exact match" />
          <SwitchGroupWide name="title" label="Title" description="Parsed title" />
          <SwitchGroupWide name="sub_title" label="Sub Title" description="Parsed Sub Title like Episode Name" />
          <SwitchGroupWide name="year" label="Year" />
          <SwitchGroupWide name="month" label="Month" description="For daily releases" />
          <SwitchGroupWide name="day" label="Day" description="For daily releases" />
          <SwitchGroupWide name="source" label="Source" />
          <SwitchGroupWide name="resolution" label="Resolution" />
          <SwitchGroupWide name="codec" label="Codec" />
          <SwitchGroupWide name="container" label="Container" />
          <SwitchGroupWide name="dynamic_range" label="Dynamic Range" />
          <SwitchGroupWide name="audio" label="Audio" />
          <SwitchGroupWide name="group" label="Group" description="Release group" />
          <SwitchGroupWide name="season" label="Season" />
          <SwitchGroupWide name="episode" label="Episode" />
          <SwitchGroupWide name="website" label="Website/Service" description="Services such as AMZN/HULU/NF" />
          <SwitchGroupWide name="proper" label="Proper" />
          <SwitchGroupWide name="repack" label="Repack" />
          <SwitchGroupWide name="edition" label="Edition" />
          <SwitchGroupWide name="hybrid" label="Hybrid version" />
          <SwitchGroupWide name="language" label="Language" />
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

          <SwitchGroupWide name="release_name" label="Release name" description="Full release name" />
          <SwitchGroupWide name="hash" label="Hash" description="Normalized hash of the release name. Use with Release name for exact match" />
          <SwitchGroupWide name="title" label="Title" description="Parsed title" />
          <SwitchGroupWide name="sub_title" label="Sub Title" description="Parsed Sub Title like Episode Name" />
          <SwitchGroupWide name="year" label="Year" />
          <SwitchGroupWide name="month" label="Month" description="For daily releases" />
          <SwitchGroupWide name="day" label="Day" description="For daily releases" />
          <SwitchGroupWide name="source" label="Source" />
          <SwitchGroupWide name="resolution" label="Resolution" />
          <SwitchGroupWide name="codec" label="Codec" />
          <SwitchGroupWide name="container" label="Container" />
          <SwitchGroupWide name="dynamic_range" label="Dynamic Range (HDR,DV etc)" />
          <SwitchGroupWide name="audio" label="Audio" />
          <SwitchGroupWide name="group" label="Group" description="Release group" />
          <SwitchGroupWide name="season" label="Season" />
          <SwitchGroupWide name="episode" label="Episode" />
          <SwitchGroupWide name="website" label="Website/Service" description="Services such as AMZN/HULU/NF" />
          <SwitchGroupWide name="repack" label="Repack" />
          <SwitchGroupWide name="proper" label="Proper" />
          <SwitchGroupWide name="edition" label="Edition and Cut" />
          <SwitchGroupWide name="hybrid" label="Hybrid version" />
          <SwitchGroupWide name="language" label="Language and Region" />
        </div>
      )}
    </SlideOver>
  );
}

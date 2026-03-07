/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { APIClient } from "@api/APIClient.ts";
import { ReleaseProfileDuplicateKeys } from "@api/query_keys.ts";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast.tsx";
import { SwitchGroupWide, TextFieldWide } from "@components/inputs/tanstack";
import { ContextField } from "@app/lib/form";
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
          <ContextField name="name">
            <TextFieldWide required label="Name"/>
          </ContextField>

          <ContextField name="release_name">
            <SwitchGroupWide label="Release name" description="Full release name" />
          </ContextField>
          <ContextField name="hash">
            <SwitchGroupWide label="Hash" description="Normalized hash of the release name. Use with Release name for exact match" />
          </ContextField>
          <ContextField name="title">
            <SwitchGroupWide label="Title" description="Parsed title" />
          </ContextField>
          <ContextField name="sub_title">
            <SwitchGroupWide label="Sub Title" description="Parsed Sub Title like Episode Name" />
          </ContextField>
          <ContextField name="year">
            <SwitchGroupWide label="Year" />
          </ContextField>
          <ContextField name="month">
            <SwitchGroupWide label="Month" description="For daily releases" />
          </ContextField>
          <ContextField name="day">
            <SwitchGroupWide label="Day" description="For daily releases" />
          </ContextField>
          <ContextField name="source">
            <SwitchGroupWide label="Source" />
          </ContextField>
          <ContextField name="resolution">
            <SwitchGroupWide label="Resolution" />
          </ContextField>
          <ContextField name="codec">
            <SwitchGroupWide label="Codec" />
          </ContextField>
          <ContextField name="container">
            <SwitchGroupWide label="Container" />
          </ContextField>
          <ContextField name="dynamic_range">
            <SwitchGroupWide label="Dynamic Range" />
          </ContextField>
          <ContextField name="audio">
            <SwitchGroupWide label="Audio" />
          </ContextField>
          <ContextField name="group">
            <SwitchGroupWide label="Group" description="Release group" />
          </ContextField>
          <ContextField name="season">
            <SwitchGroupWide label="Season" />
          </ContextField>
          <ContextField name="episode">
            <SwitchGroupWide label="Episode" />
          </ContextField>
          <ContextField name="website">
            <SwitchGroupWide label="Website/Service" description="Services such as AMZN/HULU/NF" />
          </ContextField>
          <ContextField name="proper">
            <SwitchGroupWide label="Proper" />
          </ContextField>
          <ContextField name="repack">
            <SwitchGroupWide label="Repack" />
          </ContextField>
          <ContextField name="edition">
            <SwitchGroupWide label="Edition" />
          </ContextField>
          <ContextField name="hybrid">
            <SwitchGroupWide label="Hybrid version" />
          </ContextField>
          <ContextField name="language">
            <SwitchGroupWide label="Language" />
          </ContextField>
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
          <ContextField name="name">
            <TextFieldWide required label="Name"/>
          </ContextField>

          <ContextField name="release_name">
            <SwitchGroupWide label="Release name" description="Full release name" />
          </ContextField>
          <ContextField name="hash">
            <SwitchGroupWide label="Hash" description="Normalized hash of the release name. Use with Release name for exact match" />
          </ContextField>
          <ContextField name="title">
            <SwitchGroupWide label="Title" description="Parsed title" />
          </ContextField>
          <ContextField name="sub_title">
            <SwitchGroupWide label="Sub Title" description="Parsed Sub Title like Episode Name" />
          </ContextField>
          <ContextField name="year">
            <SwitchGroupWide label="Year" />
          </ContextField>
          <ContextField name="month">
            <SwitchGroupWide label="Month" description="For daily releases" />
          </ContextField>
          <ContextField name="day">
            <SwitchGroupWide label="Day" description="For daily releases" />
          </ContextField>
          <ContextField name="source">
            <SwitchGroupWide label="Source" />
          </ContextField>
          <ContextField name="resolution">
            <SwitchGroupWide label="Resolution" />
          </ContextField>
          <ContextField name="codec">
            <SwitchGroupWide label="Codec" />
          </ContextField>
          <ContextField name="container">
            <SwitchGroupWide label="Container" />
          </ContextField>
          <ContextField name="dynamic_range">
            <SwitchGroupWide label="Dynamic Range (HDR,DV etc)" />
          </ContextField>
          <ContextField name="audio">
            <SwitchGroupWide label="Audio" />
          </ContextField>
          <ContextField name="group">
            <SwitchGroupWide label="Group" description="Release group" />
          </ContextField>
          <ContextField name="season">
            <SwitchGroupWide label="Season" />
          </ContextField>
          <ContextField name="episode">
            <SwitchGroupWide label="Episode" />
          </ContextField>
          <ContextField name="website">
            <SwitchGroupWide label="Website/Service" description="Services such as AMZN/HULU/NF" />
          </ContextField>
          <ContextField name="repack">
            <SwitchGroupWide label="Repack" />
          </ContextField>
          <ContextField name="proper">
            <SwitchGroupWide label="Proper" />
          </ContextField>
          <ContextField name="edition">
            <SwitchGroupWide label="Edition and Cut" />
          </ContextField>
          <ContextField name="hybrid">
            <SwitchGroupWide label="Hybrid version" />
          </ContextField>
          <ContextField name="language">
            <SwitchGroupWide label="Language and Region" />
          </ContextField>
        </div>
      )}
    </SlideOver>
  );
}

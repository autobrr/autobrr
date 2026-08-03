/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent
} from "@dnd-kit/core";
import {
  SortableContext,
  arrayMove,
  rectSortingStrategy,
  sortableKeyboardCoordinates,
  useSortable
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { EyeIcon, EyeSlashIcon, Squares2X2Icon } from "@heroicons/react/24/outline";

import { classNames } from "@utils";
import { DashboardConfigContext } from "@utils/Context";
import type { DashboardConfigType } from "@utils/Context";
import { DASHBOARD_WIDGETS } from "./widgets";
import type { DashboardWidgetDef } from "./widgets";

interface WidgetLayout {
  def: DashboardWidgetDef;
  hidden: boolean;
}

const resolveLayout = (config: DashboardConfigType): WidgetLayout[] => {
  const byId = new Map(DASHBOARD_WIDGETS.map((def) => [def.id, def]));
  const seen = new Set<string>();
  const layout: WidgetLayout[] = [];

  for (const entry of config.widgets) {
    const def = byId.get(entry.id);
    if (def && !seen.has(entry.id)) {
      seen.add(entry.id);
      layout.push({ def, hidden: entry.hidden });
    }
  }
  for (const def of DASHBOARD_WIDGETS) {
    if (!seen.has(def.id)) {
      layout.push({ def, hidden: false });
    }
  }

  return layout;
};

interface SortableWidgetProps {
  def: DashboardWidgetDef;
  hidden: boolean;
  editing: boolean;
  onToggle: () => void;
}

const SortableWidget = ({ def, hidden, editing, onToggle }: SortableWidgetProps) => {
  const { t } = useTranslation("common");
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: def.id,
    disabled: !editing
  });

  return (
    <div
      ref={setNodeRef}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      className={classNames(
        def.spanClass,
        isDragging ? "z-10" : "",
        editing ? "relative rounded-lg ring-2 ring-blue-400/40 dark:ring-blue-500/30 cursor-grab active:cursor-grabbing touch-none" : ""
      )}
      {...attributes}
      {...listeners}
    >
      <div className={classNames("h-full", editing ? "pointer-events-none select-none" : "", hidden ? "opacity-40" : "")}>
        <def.Component />
      </div>
      {editing && (
        <button
          type="button"
          onClick={onToggle}
          className="absolute top-2 right-2 z-10 p-1.5 rounded-md bg-white dark:bg-gray-815 border border-gray-250 dark:border-gray-750 shadow text-gray-500 hover:text-gray-800 dark:hover:text-gray-200 transition-colors duration-200 ease-in-out"
          title={hidden ? t("dashboardCustomize.show") : t("dashboardCustomize.hide")}
        >
          {hidden
            ? <EyeSlashIcon className="h-4 w-4" aria-hidden="true" />
            : <EyeIcon className="h-4 w-4" aria-hidden="true" />}
        </button>
      )}
    </div>
  );
};

export const DashboardGrid = () => {
  const { t } = useTranslation("common");
  const config = DashboardConfigContext.useValue();
  const [editing, setEditing] = useState(false);

  const layout = useMemo(() => resolveLayout(config), [config]);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  );

  const persist = (next: WidgetLayout[]) => {
    DashboardConfigContext.set({
      version: 1,
      widgets: next.map(({ def, hidden }) => ({ id: def.id, hidden }))
    });
  };

  const onDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) {
      return;
    }
    const oldIndex = layout.findIndex((l) => l.def.id === active.id);
    const newIndex = layout.findIndex((l) => l.def.id === over.id);
    if (oldIndex !== -1 && newIndex !== -1) {
      persist(arrayMove(layout, oldIndex, newIndex));
    }
  };

  const toggleWidget = (id: string) => {
    persist(layout.map((l) => (l.def.id === id ? { ...l, hidden: !l.hidden } : l)));
  };

  return (
    <div>
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold text-black dark:text-white">
          {t("dashboardStats.title")}
        </h1>
        {editing ? (
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => DashboardConfigContext.set({ version: 1, widgets: [] })}
              className="px-2.5 py-1.5 rounded-md text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors duration-200 ease-in-out"
            >
              {t("dashboardCustomize.reset")}
            </button>
            <button
              type="button"
              onClick={() => setEditing(false)}
              className="px-3 py-1.5 rounded-md bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium transition-colors duration-200 ease-in-out"
            >
              {t("dashboardCustomize.done")}
            </button>
          </div>
        ) : (
          <button
            type="button"
            onClick={() => setEditing(true)}
            className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-md text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors duration-200 ease-in-out"
          >
            <Squares2X2Icon className="h-4 w-4" aria-hidden="true" />
            {t("dashboardCustomize.customize")}
          </button>
        )}
      </div>
      {editing && (
        <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
          {t("dashboardCustomize.hint")}
        </p>
      )}

      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
        <SortableContext items={layout.map((l) => l.def.id)} strategy={rectSortingStrategy}>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-2 sm:gap-5 mt-5">
            {layout.map(({ def, hidden }) =>
              (editing || !hidden) && (
                <SortableWidget
                  key={def.id}
                  def={def}
                  hidden={hidden}
                  editing={editing}
                  onToggle={() => toggleWidget(def.id)}
                />
              )
            )}
          </div>
        </SortableContext>
      </DndContext>
    </div>
  );
};

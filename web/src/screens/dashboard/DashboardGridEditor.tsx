/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useLayoutEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  DndContext,
  DragOverlay,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  pointerWithin,
  useSensor,
  useSensors,
  type CollisionDetection,
  type DropAnimation,
  type DragEndEvent,
  type DragStartEvent
} from "@dnd-kit/core";
import {
  SortableContext,
  arrayMove,
  rectSortingStrategy,
  sortableKeyboardCoordinates,
  useSortable
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { Bars3Icon, EyeIcon, EyeSlashIcon } from "@heroicons/react/24/outline";

import { classNames } from "@utils";
import type { DashboardWidgetDef, WidgetLayout } from "./widgets";
import { WidgetBoundary } from "./WidgetBoundary";

interface SortableWidgetProps {
  def: DashboardWidgetDef;
  hidden: boolean;
  committing: boolean;
  onToggle: () => void;
}

const dropDuration = 180;
const dropEasing = "cubic-bezier(0.16, 1, 0.3, 1)";

interface WidgetPosition {
  left: number;
  top: number;
}

interface PendingDrop {
  activeId: string;
  order: string;
  positions: Map<string, WidgetPosition>;
}

const SortableWidget = ({ def, hidden, committing, onToggle }: SortableWidgetProps) => {
  const { t } = useTranslation("common");
  const { attributes, listeners, setActivatorNodeRef, setNodeRef, transform, isDragging, isSorting } = useSortable({
    id: def.id,
    data: { size: def.size },
    transition: null
  });

  return (
    <div
      ref={setNodeRef}
      data-widget-id={def.id}
      style={{ transform: CSS.Translate.toString(transform) }}
      className={classNames(
        def.spanClass,
        "h-full min-w-0 flex flex-col rounded-xl border border-blue-400/30 bg-blue-500/5 p-2 will-change-transform dark:border-blue-500/25 dark:bg-blue-500/5",
        isDragging ? "z-10 opacity-30" : "",
        isSorting && !isDragging && !committing
          ? "transition-transform duration-200 ease-out motion-reduce:transition-none"
          : "transition-none"
      )}
    >
      <div className="mb-2 flex h-9 items-center justify-between gap-2">
        <button
          ref={setActivatorNodeRef}
          type="button"
          className="inline-flex size-9 touch-none cursor-grab items-center justify-center rounded-lg text-gray-500 transition-colors duration-200 ease-in-out hover:text-gray-800 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-500 active:cursor-grabbing dark:hover:text-gray-200"
          aria-label={`${t("dashboardCustomize.move")} ${t(def.titleKey)}`}
          {...attributes}
          {...listeners}
        >
          <Bars3Icon className="size-5" aria-hidden="true" />
        </button>
        <button
          type="button"
          onClick={onToggle}
          className="inline-flex size-9 items-center justify-center rounded-lg text-gray-500 transition-colors duration-200 ease-in-out hover:bg-white hover:text-gray-800 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-500 dark:hover:bg-gray-800 dark:hover:text-gray-200 cursor-pointer"
          title={hidden ? t("dashboardCustomize.show") : t("dashboardCustomize.hide")}
          aria-label={hidden ? t("dashboardCustomize.show") : t("dashboardCustomize.hide")}
        >
          {hidden
            ? <EyeSlashIcon className="size-5" aria-hidden="true" />
            : <EyeIcon className="size-5" aria-hidden="true" />}
        </button>
      </div>
      <div className={classNames("h-full min-h-0 flex-1 select-none", hidden ? "opacity-40" : "")}>
        <WidgetBoundary def={def} />
      </div>
    </div>
  );
};

const compatibleWidgetCollisions: CollisionDetection = (args) => {
  const activeSize = args.active.data.current?.size;
  // Mixed spans do not share stable grid slots, so only equal-size widgets are valid targets.
  const droppableContainers = args.droppableContainers.filter(
    (container) => container.data.current?.size === activeSize
  );
  const compatibleArgs = { ...args, droppableContainers };
  const pointerCollisions = pointerWithin(compatibleArgs);

  return args.pointerCoordinates ? pointerCollisions : closestCenter(compatibleArgs);
};

interface DashboardGridEditorProps {
  layout: WidgetLayout[];
  persist: (next: WidgetLayout[]) => void;
}

export const DashboardGridEditor = ({ layout, persist }: DashboardGridEditorProps) => {
  const [activeId, setActiveId] = useState<string | null>(null);
  const [committing, setCommitting] = useState(false);
  const gridRef = useRef<HTMLDivElement>(null);
  const dropTargetRect = useRef<{ left: number; top: number; width: number; height: number } | null>(null);
  const pendingDrop = useRef<PendingDrop | null>(null);

  const activeDef = layout.find(({ def }) => def.id === activeId)?.def;

  const dropAnimation = useMemo<DropAnimation>(() => async ({ active, dragOverlay, transform }) => {
    const target = dropTargetRect.current;
    const view = active.node.ownerDocument.defaultView;
    if (!target || view?.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      return;
    }

    // The default overlay returns to its source; the saved target keeps the release continuous.
    const finalTransform = {
      x: transform.x - (dragOverlay.rect.left - target.left),
      y: transform.y - (dragOverlay.rect.top - target.top),
      scaleX: target.width * transform.scaleX / dragOverlay.rect.width,
      scaleY: target.height * transform.scaleY / dragOverlay.rect.height
    };
    const previousOpacity = active.node.style.opacity;
    active.node.style.opacity = "0";

    const animation = dragOverlay.node.animate([
      { transform: CSS.Transform.toString(transform) },
      { transform: CSS.Transform.toString(finalTransform) }
    ], {
      duration: dropDuration,
      easing: dropEasing,
      fill: "forwards"
    });

    await animation.finished.catch(() => undefined);
    active.node.style.opacity = previousOpacity;
    dropTargetRect.current = null;
  }, []);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  );

  useLayoutEffect(() => {
    const grid = gridRef.current;
    const pending = pendingDrop.current;
    const order = layout.map(({ def }) => def.id).join(",");
    if (!committing || !grid || !pending || pending.order !== order) {
      return;
    }

    const reduceMotion = grid.ownerDocument.defaultView?.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const animations: Animation[] = [];

    if (!reduceMotion) {
      // Persisting reorders the DOM immediately, so FLIP preserves each displaced widget's visual position.
      for (const node of grid.querySelectorAll<HTMLElement>("[data-widget-id]")) {
        const id = node.dataset.widgetId;
        const previous = id ? pending.positions.get(id) : undefined;
        if (!id || id === pending.activeId || !previous) {
          continue;
        }

        const current = node.getBoundingClientRect();
        const deltaX = previous.left - current.left;
        const deltaY = previous.top - current.top;
        if (deltaX === 0 && deltaY === 0) {
          continue;
        }

        animations.push(node.animate([
          { transform: `translate3d(${deltaX}px, ${deltaY}px, 0)` },
          { transform: "translate3d(0, 0, 0)" }
        ], {
          duration: dropDuration,
          easing: dropEasing
        }));
      }
    }

    pendingDrop.current = null;

    if (animations.length === 0) {
      setCommitting(false);
      return;
    }

    void Promise.all(animations.map((animation) => animation.finished.catch(() => undefined)))
      .then(() => setCommitting(false));
  }, [committing, layout]);

  const resetDropState = () => {
    dropTargetRect.current = null;
    pendingDrop.current = null;
    setCommitting(false);
    setActiveId(null);
  };

  const onDragStart = ({ active }: DragStartEvent) => {
    resetDropState();
    setActiveId(String(active.id));
  };

  const onDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    setActiveId(null);
    if (!over || active.id === over.id) {
      return;
    }
    const oldIndex = layout.findIndex((l) => l.def.id === active.id);
    const newIndex = layout.findIndex((l) => l.def.id === over.id);
    if (oldIndex !== -1 && newIndex !== -1 && layout[oldIndex].def.size === layout[newIndex].def.size) {
      const nextLayout = arrayMove(layout, oldIndex, newIndex);
      dropTargetRect.current = {
        left: over.rect.left,
        top: over.rect.top,
        width: over.rect.width,
        height: over.rect.height
      };
      pendingDrop.current = {
        activeId: String(active.id),
        order: nextLayout.map(({ def }) => def.id).join(","),
        positions: new Map(
          Array.from(gridRef.current?.querySelectorAll<HTMLElement>("[data-widget-id]") ?? []).map((node) => {
            const rect = node.getBoundingClientRect();
            return [node.dataset.widgetId ?? "", { left: rect.left, top: rect.top }];
          })
        )
      };
      setCommitting(true);
      persist(nextLayout);
    }
  };

  const toggleWidget = (id: string) => {
    persist(layout.map((l) => (l.def.id === id ? { ...l, hidden: !l.hidden } : l)));
  };

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={compatibleWidgetCollisions}
      onDragStart={onDragStart}
      onDragCancel={resetDropState}
      onDragEnd={onDragEnd}
    >
      <SortableContext items={layout.map((l) => l.def.id)} strategy={rectSortingStrategy}>
        <div ref={gridRef} className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-2 sm:gap-5 mt-5">
          {layout.map(({ def, hidden }) => (
            <SortableWidget
              key={def.id}
              def={def}
              hidden={hidden}
              committing={committing}
              onToggle={() => toggleWidget(def.id)}
            />
          ))}
        </div>
      </SortableContext>
      <DragOverlay dropAnimation={dropAnimation}>
        {activeDef && (
          <div className="pointer-events-none h-full w-full overflow-hidden rounded-lg shadow-2xl ring-2 ring-blue-500/50">
            <WidgetBoundary def={activeDef} />
          </div>
        )}
      </DragOverlay>
    </DndContext>
  );
};

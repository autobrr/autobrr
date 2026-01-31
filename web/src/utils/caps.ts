export type CapsCategoryItem = {
  id: number;
  name: string;
};

const toRecord = (value: unknown): Record<string, unknown> | null => {
  if (!value || typeof value !== "object") {
    return null;
  }

  return value as Record<string, unknown>;
};

const getCategoryId = (value: unknown): number | null => {
  const record = toRecord(value);
  if (!record) {
    return null;
  }

  const rawId = record.ID ?? record.id;
  if (rawId === null || typeof rawId === "undefined") {
    return null;
  }

  const parsed = Number(rawId);
  if (Number.isNaN(parsed)) {
    return null;
  }

  return parsed;
};

const getCategoryName = (value: unknown): string => {
  const record = toRecord(value);
  if (!record) {
    return "";
  }

  const rawName = record.Name ?? record.name;
  if (typeof rawName !== "string") {
    return "";
  }

  return rawName.trim();
};

const buildCategoryName = (parentName: string, name: string): string => {
  if (!parentName) {
    return name;
  }

  if (!name) {
    return parentName;
  }

  const normalizedParent = parentName.toLowerCase();
  const normalizedName = name.toLowerCase();

  if (normalizedName.startsWith(normalizedParent) || name.includes("/")) {
    return name;
  }

  return `${parentName} / ${name}`;
};

const getSubCategories = (value: unknown): unknown[] => {
  const record = toRecord(value);
  if (!record) {
    return [];
  }

  const raw = record.SubCategories ?? record.subCategories ?? record.subcategories;
  return Array.isArray(raw) ? raw : [];
};

const getCategoriesList = (caps: Record<string, unknown>): unknown[] => {
  const direct = caps.categories ?? caps.Categories;
  if (Array.isArray(direct)) {
    return direct;
  }

  const wrapped = toRecord(caps.Categories ?? caps.categories);
  if (!wrapped) {
    return [];
  }

  const wrappedList = wrapped.Categories ?? wrapped.categories;
  return Array.isArray(wrappedList) ? wrappedList : [];
};

export const extractCategoriesFromCaps = (caps: unknown): CapsCategoryItem[] => {
  const root = toRecord(caps);
  if (!root) {
    return [];
  }

  const categories = getCategoriesList(root);
  if (!Array.isArray(categories) || categories.length === 0) {
    return [];
  }

  const items: CapsCategoryItem[] = [];
  const seen = new Set<number>();

  const visit = (category: unknown, parentName = "") => {
    const id = getCategoryId(category);
    const name = getCategoryName(category);
    const displayName = buildCategoryName(parentName, name);

    if (id !== null && displayName && !seen.has(id)) {
      items.push({ id, name: displayName });
      seen.add(id);
    }

    const subCategories = getSubCategories(category);
    subCategories.forEach((subCategory) => {
      visit(subCategory, name || parentName);
    });
  };

  categories.forEach((category) => visit(category));

  return items.sort((a, b) => a.name.localeCompare(b.name));
};

export const parseCapabilitiesPayload = (capabilities?: unknown): unknown | null => {
  if (!capabilities) {
    return null;
  }

  if (typeof capabilities === "string") {
    try {
      return JSON.parse(capabilities);
    } catch {
      return null;
    }
  }

  return capabilities;
};

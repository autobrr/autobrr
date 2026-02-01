export type CapsCategoryItem = {
  id: number;
  name: string;
};

export type CapsCategoryNode = {
  id: number;
  name: string;
  subcategories: CapsCategoryNode[];
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
  return flattenCategories(extractCategoryTreeFromCaps(caps));
};

export const extractCategoryTreeFromCaps = (caps: unknown): CapsCategoryNode[] => {
  const root = toRecord(caps);
  if (!root) {
    return [];
  }

  const categories = getCategoriesList(root);
  if (!Array.isArray(categories) || categories.length === 0) {
    return [];
  }

  const seen = new Set<number>();

  const buildNode = (category: unknown, parentName = ""): CapsCategoryNode | null => {
    const id = getCategoryId(category);
    const name = getCategoryName(category);
    if (id === null || !name) {
      return null;
    }

    if (seen.has(id)) {
      return null;
    }

    seen.add(id);

    const subCategories = getSubCategories(category)
      .map((subCategory) => buildNode(subCategory, name || parentName))
      .filter((subCategory): subCategory is CapsCategoryNode => Boolean(subCategory));

    return {
      id,
      name: parentName ? buildCategoryName(parentName, name) : name,
      subcategories: subCategories
    };
  };

  return categories
    .map((category) => buildNode(category))
    .filter((category): category is CapsCategoryNode => Boolean(category));
};

export const flattenCategoryIds = (nodes: CapsCategoryNode[]): number[] => {
  const ids: number[] = [];
  nodes.forEach((node) => {
    ids.push(node.id);
    if (node.subcategories.length) {
      ids.push(...flattenCategoryIds(node.subcategories));
    }
  });
  return ids;
};

const flattenCategories = (nodes: CapsCategoryNode[]): CapsCategoryItem[] => {
  const items: CapsCategoryItem[] = [];
  nodes.forEach((node) => {
    items.push({ id: node.id, name: node.name });
    if (node.subcategories.length) {
      items.push(...flattenCategories(node.subcategories));
    }
  });
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

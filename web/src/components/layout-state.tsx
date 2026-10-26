"use client";

import { createContext, useCallback, useContext, useMemo, useState, useSyncExternalStore, type ReactNode } from "react";

// What the header and sidebar share with the pages: whether the sidebar is shown, and the topic whose title the
// header shows once its own title has scrolled away. On desktop the sidebar is a column that can be hidden; on
// phones it is a panel over the page, closed until the menu button opens it.
interface HeaderTopic {
  id: number;
  title: string;
  url: string;
  categoryId: number;
}

interface LayoutState {
  sidebarVisible: boolean;
  mobileSidebarOpen: boolean;
  toggleSidebar: () => void;
  hideSidebarOnMobile: () => void;
  headerTopic: HeaderTopic | null;
  setHeaderTopic: (topic: HeaderTopic | null) => void;
}

const LayoutContext = createContext<LayoutState | null>(null);

const isMobile = () => window.innerWidth < 768;

// whether the desktop sidebar was hidden, remembered in the browser; the server renders it shown
const hiddenKey = "casnode-sidebar-hidden";
const hiddenEvent = "casnode-sidebar-hidden-change";

function readHidden(): boolean {
  try {
    return localStorage.getItem(hiddenKey) === "true";
  } catch {
    return false;
  }
}

function subscribeHidden(callback: () => void) {
  window.addEventListener(hiddenEvent, callback);
  window.addEventListener("storage", callback);
  return () => {
    window.removeEventListener(hiddenEvent, callback);
    window.removeEventListener("storage", callback);
  };
}

export function LayoutStateProvider({ children }: { children: ReactNode }) {
  const sidebarVisible = !useSyncExternalStore(subscribeHidden, readHidden, () => false);
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);
  const [headerTopic, setHeaderTopic] = useState<HeaderTopic | null>(null);

  const toggleSidebar = useCallback(() => {
    if (isMobile()) {
      setMobileSidebarOpen((open) => !open);
      return;
    }
    try {
      localStorage.setItem(hiddenKey, String(!readHidden()));
    } catch {
      // the choice is just not remembered
    }
    window.dispatchEvent(new Event(hiddenEvent));
  }, []);

  const hideSidebarOnMobile = useCallback(() => setMobileSidebarOpen(false), []);

  const value = useMemo(
    () => ({ sidebarVisible, mobileSidebarOpen, toggleSidebar, hideSidebarOnMobile, headerTopic, setHeaderTopic }),
    [sidebarVisible, mobileSidebarOpen, toggleSidebar, hideSidebarOnMobile, headerTopic],
  );
  return <LayoutContext.Provider value={value}>{children}</LayoutContext.Provider>;
}

export function useLayoutState(): LayoutState {
  const value = useContext(LayoutContext);
  if (!value) {
    throw new Error("useLayoutState() outside LayoutStateProvider");
  }
  return value;
}

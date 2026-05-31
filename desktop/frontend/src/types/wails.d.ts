import type { WailsDesktopApi, WailsRuntime } from ".";

declare global {
  interface Window {
    go?: {
      mainapp?: {
        App?: WailsDesktopApi;
      };
    };
    runtime?: WailsRuntime;
  }
}

export {};

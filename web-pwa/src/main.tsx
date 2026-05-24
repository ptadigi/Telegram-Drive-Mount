import React from "react";
import ReactDOM from "react-dom/client";
import "./i18n";
import "./styles.css";
import { App } from "./App";
import { UIProvider } from "./state/ui";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <UIProvider>
      <App />
    </UIProvider>
  </React.StrictMode>,
);

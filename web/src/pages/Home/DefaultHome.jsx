import React, { useState, useEffect, useCallback } from 'react';
import { getServerAddress } from '../../helpers/token';
import Sidebar from './components/Sidebar';
import IntroPage from './pages/IntroPage';
import AdvantagePage from './pages/AdvantagePage';
import StartPage from './pages/StartPage';
import ConfigPage from './pages/ConfigPage';
import DowngradePage from './pages/DowngradePage';
import VscodePage from './pages/VscodePage';
import ErrorsPage from './pages/ErrorsPage';
import ApiPage from './pages/ApiPage';
import FaqPage from './pages/FaqPage';
import CommandsPage from './pages/CommandsPage';

const PAGES = {
  intro: IntroPage,
  advantage: AdvantagePage,
  start: StartPage,
  config: ConfigPage,
  downgrade: DowngradePage,
  vscode: VscodePage,
  errors: ErrorsPage,
  api: ApiPage,
  faq: FaqPage,
  commands: CommandsPage,
};

const getHashPage = () => {
  const hash = window.location.hash.replace('#', '');
  return PAGES[hash] ? hash : 'intro';
};

const OFFICIAL_URL = 'https://rabbitcode.cc';

const getDisplayAddress = () => {
  const addr = getServerAddress();
  if (/^https?:\/\/(localhost|127\.0\.0\.1)(:\d+)?/.test(addr)) {
    return OFFICIAL_URL;
  }
  return addr;
};

const DefaultHome = () => {
  const [activePage, setActivePage] = useState(getHashPage);
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const serverAddress = getDisplayAddress();

  const navigate = useCallback((id) => {
    setActivePage(id);
    window.location.hash = id;
  }, []);

  useEffect(() => {
    const onHash = () => setActivePage(getHashPage());
    window.addEventListener('hashchange', onHash);
    return () => window.removeEventListener('hashchange', onHash);
  }, []);

  useEffect(() => {
    const onKey = (e) => {
      if (e.key === 'Escape') setSidebarOpen(false);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  const PageComponent = PAGES[activePage] || IntroPage;

  return (
    <>
      <div className="hp-scene" aria-hidden="true" />
      <div className="hp-layout">
        <Sidebar
          activePage={activePage}
          onNavigate={navigate}
          open={sidebarOpen}
          onClose={() => setSidebarOpen(false)}
        />
        <button
          className={`hp-mobile-menu${sidebarOpen ? ' open' : ''}`}
          onClick={() => setSidebarOpen((v) => !v)}
          type="button"
          aria-label="Menu"
        >
          <span className="hp-menu-toggle-icon" aria-hidden="true" />
        </button>
        <main className="hp-main">
          <div className="hp-main-inner">
            <PageComponent key={activePage} serverAddress={serverAddress} />
          </div>
        </main>
      </div>
      <div
        className={`hp-overlay${sidebarOpen ? ' active' : ''}`}
        onClick={() => setSidebarOpen(false)}
        aria-hidden="true"
      />
    </>
  );
};

export default DefaultHome;

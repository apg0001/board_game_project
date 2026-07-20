import {
  Bell,
  Bot,
  ChevronRight,
  Clock3,
  Crown,
  Dice5,
  Flame,
  Gamepad2,
  LockKeyhole,
  MessageCircle,
  Play,
  Plus,
  Search,
  ShieldCheck,
  Sparkles,
  Trophy,
  UsersRound,
  Wifi
} from "lucide-react";
import type { CSSProperties } from "react";
import { useEffect, useMemo, useState } from "react";

type GameCategory = "블러핑" | "전략" | "순발력" | "숫자/조합" | "파티" | "카드";

interface GameCard {
  id: string;
  title: string;
  players: string;
  minPlayers: number;
  maxPlayers: number;
  time: string;
  difficulty: "쉬움" | "보통" | "어려움";
  categories: GameCategory[];
  accent: string;
}

const games: GameCard[] = [
  {
    id: "dalmuti",
    title: "위대한 달무티",
    players: "4-8명",
    minPlayers: 4,
    maxPlayers: 8,
    time: "20분",
    difficulty: "쉬움",
    categories: ["카드", "파티"],
    accent: "#c95f42"
  },
  {
    id: "werewolf",
    title: "한 밤의 늑대인간",
    players: "3-10명",
    minPlayers: 3,
    maxPlayers: 10,
    time: "10분",
    difficulty: "보통",
    categories: ["블러핑", "파티"],
    accent: "#6f5fbf"
  },
  {
    id: "rummikub",
    title: "루미큐브",
    players: "2-4명",
    minPlayers: 2,
    maxPlayers: 4,
    time: "35분",
    difficulty: "보통",
    categories: ["숫자/조합", "전략"],
    accent: "#2f78b7"
  },
  {
    id: "bang",
    title: "뱅!",
    players: "4-7명",
    minPlayers: 4,
    maxPlayers: 7,
    time: "40분",
    difficulty: "어려움",
    categories: ["블러핑", "카드"],
    accent: "#a3682a"
  },
  {
    id: "davinci",
    title: "다빈치 코드",
    players: "2-4명",
    minPlayers: 2,
    maxPlayers: 4,
    time: "15분",
    difficulty: "쉬움",
    categories: ["숫자/조합", "전략"],
    accent: "#1c8c8c"
  },
  {
    id: "halli-galli",
    title: "할리갈리",
    players: "2-6명",
    minPlayers: 2,
    maxPlayers: 6,
    time: "10분",
    difficulty: "쉬움",
    categories: ["순발력", "파티"],
    accent: "#d24f6a"
  },
  {
    id: "splendor",
    title: "스플랜더",
    players: "2-4명",
    minPlayers: 2,
    maxPlayers: 4,
    time: "30분",
    difficulty: "보통",
    categories: ["전략", "카드"],
    accent: "#2f8e74"
  }
];

const playerCount = 4;
const fallbackRecommendedGames = games.filter(
  (game) => game.minPlayers <= playerCount && game.maxPlayers >= playerCount
);

interface ApiGame {
  id: string;
  title: string;
  minPlayers: number;
  maxPlayers: number;
  estimatedMinutes: number;
  difficulty: "쉬움" | "보통" | "어려움";
  categories: GameCategory[];
}

interface GuestSession {
  sessionToken: string;
  user: {
    id: string;
    nickname: string;
  };
}

const guestStorageKey = "board-table.guest-session";

export function App() {
  const [apiGames, setApiGames] = useState<ApiGame[]>([]);
  const [serverStatus, setServerStatus] = useState<"연결됨" | "오프라인 모드">("오프라인 모드");
  const [guestSession, setGuestSession] = useState<GuestSession | null>(() => readGuestSession());

  useEffect(() => {
    const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";

    fetch(`${apiURL}/api/games/recommend?players=${playerCount}`)
      .then((response) => {
        if (!response.ok) throw new Error("recommendation request failed");
        return response.json() as Promise<{ games: ApiGame[] }>;
      })
      .then((data) => {
        setApiGames(data.games);
        setServerStatus("연결됨");
      })
      .catch(() => {
        setApiGames([]);
        setServerStatus("오프라인 모드");
      });
  }, []);

  useEffect(() => {
    const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";
    const current = readGuestSession();

    if (current?.sessionToken) {
      fetch(`${apiURL}/api/me`, {
        headers: { Authorization: `Bearer ${current.sessionToken}` }
      })
        .then((response) => {
          if (!response.ok) throw new Error("guest session expired");
          return response.json() as Promise<{ user: GuestSession["user"] }>;
        })
        .then((data) => {
          const refreshed = { sessionToken: current.sessionToken, user: data.user };
          saveGuestSession(refreshed);
          setGuestSession(refreshed);
        })
        .catch(() => createGuest(apiURL).then(setGuestSession).catch(() => undefined));
      return;
    }

    createGuest(apiURL).then(setGuestSession).catch(() => undefined);
  }, []);

  const recommendedGames = useMemo(() => {
    if (apiGames.length === 0) return fallbackRecommendedGames;

    return apiGames.map((apiGame) => {
      const fallback = games.find((game) => game.id === apiGame.id);
      return {
        id: apiGame.id,
        title: apiGame.title,
        players: `${apiGame.minPlayers}-${apiGame.maxPlayers}명`,
        minPlayers: apiGame.minPlayers,
        maxPlayers: apiGame.maxPlayers,
        time: `${apiGame.estimatedMinutes}분`,
        difficulty: apiGame.difficulty,
        categories: apiGame.categories,
        accent: fallback?.accent ?? "#2f8e74"
      };
    });
  }, [apiGames]);

  return (
    <main className="app-shell">
      <section className="hero-panel" aria-label="게임 로비">
        <nav className="top-bar">
          <div className="brand">
            <span className="brand-mark">
              <Dice5 size={22} aria-hidden="true" />
            </span>
            <div>
              <strong>Board Table</strong>
              <span>PWA 로비</span>
            </div>
          </div>
          <div className="top-actions">
            <button className="icon-button" aria-label="알림">
              <Bell size={19} />
            </button>
            <button className="profile-button" aria-label="내 프로필">
              {guestSession?.user.nickname.slice(-2) ?? "G"}
            </button>
          </div>
        </nav>

        <div className="hero-grid">
          <div className="hero-copy">
            <p className="eyebrow">
              <Wifi size={16} aria-hidden="true" />
              모바일, 태블릿, 데스크톱 동시 플레이 · 서버 {serverStatus}
            </p>
            <h1>친구와 바로 모이고, 인원에 맞는 게임을 즉시 시작하세요.</h1>
            <p className="summary">
              방 코드, 퀵매치, 재접속, 관전, 채팅까지 하나의 앱 화면에서 이어지는 보드게임 플랫폼입니다.
            </p>
            <div className="hero-actions">
              <button className="primary-button">
                <Play size={18} />
                퀵매치
              </button>
              <button className="secondary-button">
                <Plus size={18} />
                방 만들기
              </button>
            </div>
          </div>

          <div className="table-preview" aria-label="진행 중인 방 미리보기">
            <div className="table-felt">
              <span className="seat seat-a">민</span>
              <span className="seat seat-b">J</span>
              <span className="seat seat-c">S</span>
              <span className="seat seat-d">게</span>
              <div className="card-stack">
                <span />
                <span />
                <span />
              </div>
              <button className="bell-button" aria-label="할리갈리 종 치기">
                <Flame size={24} />
              </button>
            </div>
          </div>
        </div>
      </section>

      <section className="content-grid">
        <aside className="control-panel" aria-label="매칭 패널">
          <div className="room-card">
            <div className="section-title">
              <div>
                <span>{guestSession?.user.nickname ?? "게스트 준비 중"}</span>
                <h2>{playerCount}명 입장 중</h2>
              </div>
              <UsersRound size={22} />
            </div>
            <div className="party-list">
              {["Guest_8391", "Nara", "Min", "Seo"].map((name, index) => (
                <div className="party-member" key={name}>
                  <span>{name.slice(0, 1)}</span>
                  <div>
                    <strong>{name}</strong>
                    <small>{index === 0 ? "방장" : "준비 완료"}</small>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="room-card">
            <div className="section-title">
              <div>
                <span>방 코드</span>
                <h2>DK-4821</h2>
              </div>
              <LockKeyhole size={22} />
            </div>
            <button className="wide-button">
              초대 코드 공유
              <ChevronRight size={18} />
            </button>
          </div>

          <div className="status-strip">
            <span>
              <ShieldCheck size={17} />
              재접속 60초 보장
            </span>
            <span>
              <MessageCircle size={17} />
              로비 채팅 활성
            </span>
          </div>
        </aside>

        <section className="games-panel" aria-label="추천 게임">
          <div className="panel-heading">
            <div>
              <span className="eyebrow compact">
                <Sparkles size={15} />
                인원 기반 추천
              </span>
              <h2>{playerCount}명이 바로 플레이 가능한 게임</h2>
            </div>
            <label className="search-box">
              <Search size={17} aria-hidden="true" />
              <input type="search" placeholder="게임 검색" aria-label="게임 검색" />
            </label>
          </div>

          <div className="filter-row" aria-label="카테고리 필터">
            {["전체", "전략", "블러핑", "순발력", "숫자/조합"].map((category) => (
              <button className={category === "전체" ? "active" : ""} key={category}>
                {category}
              </button>
            ))}
          </div>

          <div className="game-grid">
            {recommendedGames.map((game) => (
              <article
                className="game-card"
                key={game.id}
                style={{ "--accent": game.accent } as CSSProperties}
              >
                <div className="game-art" aria-hidden="true">
                  <span className="token one" />
                  <span className="token two" />
                  <span className="mini-card" />
                </div>
                <div className="game-card-body">
                  <div>
                    <h3>{game.title}</h3>
                    <p>{game.categories.join(" · ")}</p>
                  </div>
                  <dl className="game-meta">
                    <div>
                      <UsersRound size={15} />
                      <dt>인원</dt>
                      <dd>{game.players}</dd>
                    </div>
                    <div>
                      <Clock3 size={15} />
                      <dt>시간</dt>
                      <dd>{game.time}</dd>
                    </div>
                    <div>
                      <Trophy size={15} />
                      <dt>난이도</dt>
                      <dd>{game.difficulty}</dd>
                    </div>
                  </dl>
                </div>
                <button className="game-action" aria-label={`${game.title} 선택`}>
                  선택
                  <ChevronRight size={17} />
                </button>
              </article>
            ))}
          </div>
        </section>
      </section>

      <section className="dock" aria-label="모바일 빠른 실행">
        <button>
          <Gamepad2 size={20} />
          로비
        </button>
        <button>
          <Bot size={20} />
          튜토리얼
        </button>
        <button className="dock-primary">
          <Play size={20} />
          시작
        </button>
        <button>
          <Crown size={20} />
          랭킹
        </button>
      </section>
    </main>
  );
}

function readGuestSession(): GuestSession | null {
  try {
    const raw = localStorage.getItem(guestStorageKey);
    return raw ? (JSON.parse(raw) as GuestSession) : null;
  } catch {
    return null;
  }
}

function saveGuestSession(session: GuestSession) {
  localStorage.setItem(guestStorageKey, JSON.stringify(session));
}

async function createGuest(apiURL: string): Promise<GuestSession> {
  const response = await fetch(`${apiURL}/api/guests`, { method: "POST" });
  if (!response.ok) throw new Error("failed to create guest");

  const session = (await response.json()) as GuestSession;
  saveGuestSession(session);
  return session;
}

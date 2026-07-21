import type { GameCard } from "./types";

export const games: GameCard[] = [
  {
    id: "dalmuti",
    title: "위대한 달무티",
    players: "4-8명",
    minPlayers: 4,
    maxPlayers: 8,
    time: "20분",
    difficulty: "쉬움",
    categories: ["카드", "파티"],
    accent: "#c95f42",
    cover: "DALMUTI",
    mark: "1"
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
    accent: "#6f5fbf",
    cover: "WEREWOLF",
    mark: "夜"
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
    accent: "#2f78b7",
    cover: "RUMMIKUB",
    mark: "30"
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
    accent: "#a3682a",
    cover: "BANG!",
    mark: "!"
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
    accent: "#1c8c8c",
    cover: "DAVINCI",
    mark: "?"
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
    accent: "#d24f6a",
    cover: "HALLI",
    mark: "5"
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
    accent: "#2f8e74",
    cover: "SPLENDOR",
    mark: "15"
  },
  {
    id: "sutda",
    title: "섯다",
    players: "2-10명",
    minPlayers: 2,
    maxPlayers: 10,
    time: "8분",
    difficulty: "보통",
    categories: ["블러핑", "카드"],
    accent: "#8f3f71",
    cover: "SUTDA",
    mark: "38"
  },
  {
    id: "gostop",
    title: "고스톱",
    players: "2-3명",
    minPlayers: 2,
    maxPlayers: 3,
    time: "20분",
    difficulty: "보통",
    categories: ["전략", "카드"],
    accent: "#b28a22",
    cover: "GO-STOP",
    mark: "光"
  },
  {
    id: "onecard",
    title: "원카드",
    players: "2-6명",
    minPlayers: 2,
    maxPlayers: 6,
    time: "10분",
    difficulty: "쉬움",
    categories: ["카드", "파티"],
    accent: "#2c7a9b",
    cover: "ONE CARD",
    mark: "A"
  },
  {
    id: "jokerdraw",
    title: "조커뽑기",
    players: "2-8명",
    minPlayers: 2,
    maxPlayers: 8,
    time: "8분",
    difficulty: "쉬움",
    categories: ["카드", "파티"],
    accent: "#7a4fb0",
    cover: "JOKER",
    mark: "J"
  }
];

export const playerCount = 4;

export const fallbackRecommendedGames = games.filter(
  (game) => game.minPlayers <= playerCount && game.maxPlayers >= playerCount
);

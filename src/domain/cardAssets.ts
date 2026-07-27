import type { HandCard } from "./types";

const rankAssetName: Record<string, string> = {
  A: "ace",
  J: "jack",
  Q: "queen",
  K: "king",
  JOKER: "joker"
};

const copySuit: Record<string, string> = {
  "1": "spades",
  "2": "hearts",
  "3": "diamonds",
  "4": "clubs"
};

const suitAssetName: Record<string, string> = {
  spade: "spades",
  heart: "hearts",
  diamond: "diamonds",
  club: "clubs"
};

export const playingCardBackAsset = "/assets/cards/playing/back.png";

export function playingCardAsset(card: HandCard) {
  if (card.joker || card.rank === "JOKER") {
    return card.id?.includes("red") || card.id?.includes("color")
      ? "/assets/cards/playing/red_joker.svg"
      : "/assets/cards/playing/black_joker.svg";
  }

  const rank = rankToAssetName(card.rank);
  const suit = suitToAssetName(card);
  if (!rank || !suit) return "";

  return `/assets/cards/playing/${rank}_of_${suit}.svg`;
}

function rankToAssetName(rank: HandCard["rank"]) {
  if (rank === undefined || rank === null) return "";
  const normalized = String(rank).toUpperCase();
  return rankAssetName[normalized] ?? normalized.toLowerCase();
}

function suitToAssetName(card: HandCard) {
  if (card.suit && suitAssetName[card.suit]) {
    return suitAssetName[card.suit];
  }
  const parts = card.id?.split("-") ?? [];
  const copyIndex = parts.length > 0 ? parts[parts.length - 1] : "";
  return copySuit[copyIndex] ?? "spades";
}

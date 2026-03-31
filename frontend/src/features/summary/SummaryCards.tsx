import type { SavingsSummary } from "../../types";

interface SummaryCardsProps {
  summary: SavingsSummary | null;
}

export function SummaryCards({ summary }: SummaryCardsProps) {
  const cards = [
    {
      label: "Monthly spend",
      value: summary ? `$${summary.totalMonthlySpend.toFixed(0)}` : "--",
    },
    {
      label: "Savings opportunity",
      value: summary ? `$${summary.totalMonthlySavings.toFixed(0)}` : "--",
    },
    {
      label: "Actionable items",
      value: summary ? String(summary.actionableCount) : "--",
    },
    {
      label: "Advisory items",
      value: summary ? String(summary.advisoryCount) : "--",
    },
  ];

  return (
    <section className="summary-grid">
      {cards.map((card) => (
        <article key={card.label} className="card">
          <span className="label">{card.label}</span>
          <strong>{card.value}</strong>
        </article>
      ))}
    </section>
  );
}

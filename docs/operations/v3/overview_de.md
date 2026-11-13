# v3-Dogus

v3-Dogus werden als Helm-Chart ausgeliefert und vom Dogu-Operator installiert.
Wie bei v2-Dogus beschreibt eine Dogu-Ressource, welches Dogu in welcher Version installiert sein soll.

Für Administratoren ändert sich vor allem:

- **Installation**: Die Dogu-Ressource wird mit `apiVersion: k8s.cloudogu.com/v3beta1` und `spec.doguApiVersion: v3` angelegt.
- **Konfiguration**: Statt Dogu-Config-Keys werden Helm-Values an der Dogu-Ressource gesetzt (`spec.values`, `spec.mappedValues`).
- **Status**: Der Zustand wird über Conditions wie `Ready` und `Healthy` gemeldet.

## Inhalt

- [v3-Dogus installieren](installing_v3_dogus_de.md): Voraussetzungen, Installation und Prüfen der Installation
- [v3-Dogus konfigurieren](configuring_v3_dogus_de.md): Values, Mapped Values, Log-Level, globale Konfiguration, Validierung und Neustartverhalten
- [v3-Dogus stoppen, starten und neu starten](stopping_starting_and_restarting_v3_dogus_de.md): Stoppen, Starten, Pausieren und Neustart
- [Conditions und Events von v3-Dogus](conditions_and_events_de.md): Referenz aller Conditions und Events
- [Migration von v2](migrating_from_v2_to_v3_de.md): Umstellung eines installierten v2-Dogus auf v3

Die Entwicklung von v3-Dogus beschreiben die [Dogu-Development-Docs](https://github.com/cloudogu/dogu-development-docs/blob/main/docs/v3/README_de.md).

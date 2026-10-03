//go:build windows

package main

import "fmt"

func texts(lang int) stringsTable {
	if !validLanguage(lang) {
		logEvent("ERROR", "localization", fmt.Sprintf("invalid language index=%d; falling back to English", lang))
		lang = langEnglish
	}
	table := []stringsTable{
		{style: "Style", normal: "Normal", opaque: "Opaque", clear: "Clear", blur: "Blur", acrylic: "Acrylic", openBoot: "Open at boot", refresh: "Refresh taskbar", checkUpdates: "Check for updates", autoUpdates: "Automatic updates", language: "Language", english: "English", french: "French", spanish: "Spanish", german: "German", russian: "Russian", about: "About", exit: "Exit", aboutBody: "LowBar v" + buildVersion + "\n\nA lightweight Windows taskbar appearance utility.\nNo telemetry. Update checks contact only the official GitHub release service.", aboutTitle: "About LowBar"},
		{style: "Style", normal: "Normal", opaque: "Opaque", clear: "Clair", blur: "Flou", acrylic: "Acrylique", openBoot: "Lancer au démarrage", refresh: "Actualiser la barre des tâches", checkUpdates: "Rechercher les mises à jour", autoUpdates: "Mises à jour automatiques", language: "Langue", english: "Anglais", french: "Français", spanish: "Espagnol", german: "Allemand", russian: "Russe", about: "À propos", exit: "Quitter", aboutBody: "LowBar\n\nUtilitaire léger pour l’apparence de la barre des tâches Windows.\nAucune télémétrie. Les mises à jour contactent uniquement le service de releases GitHub officiel.", aboutTitle: "À propos de LowBar"},
		{style: "Estilo", normal: "Normal", opaque: "Opaco", clear: "Transparente", blur: "Desenfoque", acrylic: "Acrílico", openBoot: "Abrir al iniciar", refresh: "Actualizar barra de tareas", checkUpdates: "Buscar actualizaciones", autoUpdates: "Actualizaciones automáticas", language: "Idioma", english: "Inglés", french: "Francés", spanish: "Español", german: "Alemán", russian: "Ruso", about: "Acerca de", exit: "Salir", aboutBody: "LowBar\n\nUtilidad ligera para la apariencia de la barra de tareas de Windows.\nSin telemetría. Las actualizaciones solo contactan el servicio oficial de releases de GitHub.", aboutTitle: "Acerca de LowBar"},
		{style: "Stil", normal: "Normal", opaque: "Deckend", clear: "Klar", blur: "Unschärfe", acrylic: "Acryl", openBoot: "Beim Start öffnen", refresh: "Taskleiste aktualisieren", checkUpdates: "Nach Updates suchen", autoUpdates: "Automatische Updates", language: "Sprache", english: "Englisch", french: "Französisch", spanish: "Spanisch", german: "Deutsch", russian: "Russisch", about: "Über", exit: "Beenden", aboutBody: "LowBar\n\nLeichtes Dienstprogramm für das Erscheinungsbild der Windows-Taskleiste.\nKeine Telemetrie. Updates kontaktieren nur den offiziellen GitHub-Release-Dienst.", aboutTitle: "Über LowBar"},
		{style: "Стиль", normal: "Обычный", opaque: "Непрозрачный", clear: "Прозрачный", blur: "Размытие", acrylic: "Акрил", openBoot: "Запускать при входе", refresh: "Обновить панель задач", checkUpdates: "Проверить обновления", autoUpdates: "Автоматические обновления", language: "Язык", english: "Английский", french: "Французский", spanish: "Испанский", german: "Немецкий", russian: "Русский", about: "О программе", exit: "Выход", aboutBody: "LowBar\n\nЛёгкая утилита для оформления панели задач Windows.\nБез телеметрии. Обновления обращаются только к официальному сервису релизов GitHub.", aboutTitle: "О программе LowBar"},
	}
	return table[lang]
}

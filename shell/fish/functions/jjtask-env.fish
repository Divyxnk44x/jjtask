function jjtask-env
    if test "$argv[1]" = off
        if set -q JJTASK_PROFILE
            set -l jjtask_bin "$JJTASK_PROFILE/bin"
            set -l new_path
            for p in $PATH
                if test "$p" != "$jjtask_bin"
                    set -a new_path $p
                end
            end
            set -gx PATH $new_path
            set -e JJTASK_PROFILE
            set -e JJ_CONFIG
            echo "jjtask environment deactivated"
        else
            echo "jjtask environment not active"
        end
        return
    end

    # Use JJTASK_PROFILE if already set, otherwise auto-detect
    set -l jjtask_path ""
    if set -q JJTASK_PROFILE; and test -d "$JJTASK_PROFILE/bin"
        set jjtask_path "$JJTASK_PROFILE"
    else
        for candidate in \
            (dirname (status --current-filename) 2>/dev/null | string replace '/shell/fish/functions' '') \
            ~/Projects/jjtask \
            ~/.local/share/jjtask \
            /opt/jjtask
            if test -d "$candidate/bin"
                set jjtask_path (realpath "$candidate")
                break
            end
        end
    end

    if test -z "$jjtask_path"
        echo "Error: Could not find jjtask installation"
        return 1
    end

    set -gx JJTASK_PROFILE "$jjtask_path"
    # Layer agent.toml on top of user config (: separator, later wins)
    set -l user_config (set -q XDG_CONFIG_HOME; and echo "$XDG_CONFIG_HOME/jj/config.toml"; or echo "$HOME/.config/jj/config.toml")
    if test -f "$user_config"
        set -gx JJ_CONFIG "$user_config:$jjtask_path/config/agent.toml"
    else
        set -gx JJ_CONFIG "$jjtask_path/config/agent.toml"
    end
    set -gx PATH $jjtask_path/bin $PATH

    # Symlink completions if not present
    set -l fish_comp_dir (set -q XDG_CONFIG_HOME; and echo "$XDG_CONFIG_HOME/fish/completions"; or echo "$HOME/.config/fish/completions")
    if not test -e "$fish_comp_dir/jjtask.fish"
        mkdir -p "$fish_comp_dir"
        ln -s "$jjtask_path/shell/fish/completions/jjtask.fish" "$fish_comp_dir/jjtask.fish"
        echo "  Completions symlinked to $fish_comp_dir/jjtask.fish"
    end

    echo "jjtask environment loaded from $jjtask_path"
end
